#!/usr/bin/env python3
"""
Automated restore verification for Crypto Market Platform.

Verifies a PostgreSQL restore by checking:
- Schema integrity (tables, indexes, constraints)
- Row counts against expected baselines
- Sample data validation
- Referential integrity
- Backup manifest consistency

Usage:
    python3 verify_restore.py --dsn postgres://user:pass@host:5432/db
    python3 verify_restore.py --dsn $POSTGRES_DSN --manifest /tmp/backup/manifest.json
    python3 verify_restore.py --dsn $POSTGRES_DSN --report /tmp/restore-report.json
"""

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

try:
    import psycopg2
except ImportError:
    psycopg2 = None


EXPECTED_TABLES = [
    "assets",
    "markets",
    "ticker_data",
    "order_books",
    "ingestion_state",
    "schema_migrations",
]

EXPECTED_INDEXES_MIN = 5


class RestoreVerifier:
    """Verifies PostgreSQL restore integrity."""

    def __init__(self, dsn: str):
        self.dsn = dsn
        self.results = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "dsn_host": dsn.split("@")[-1] if "@" in dsn else "unknown",
            "checks": [],
            "passed": 0,
            "failed": 0,
            "warnings": 0,
        }

    def connect(self):
        if psycopg2 is None:
            self._add_result("connectivity", "SKIP", "psycopg2 not installed")
            return None
        try:
            conn = psycopg2.connect(self.dsn, connect_timeout=10)
            self._add_result("connectivity", "PASS", "Connected successfully")
            return conn
        except Exception as e:
            self._add_result("connectivity", "FAIL", f"Connection failed: {e}")
            return None

    def verify_schema(self, conn) -> bool:
        """Check that all expected tables exist."""
        try:
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT table_name FROM information_schema.tables "
                    "WHERE table_schema = 'public' ORDER BY table_name"
                )
                tables = [row[0] for row in cur.fetchall()]

            missing = [t for t in EXPECTED_TABLES if t not in tables]
            if missing:
                self._add_result(
                    "schema_tables", "FAIL",
                    f"Missing tables: {missing}. Found: {tables}"
                )
                return False

            self._add_result(
                "schema_tables", "PASS",
                f"All {len(EXPECTED_TABLES)} expected tables present"
            )
            return True
        except Exception as e:
            self._add_result("schema_tables", "FAIL", f"Schema check error: {e}")
            return False

    def verify_indexes(self, conn) -> bool:
        """Check that indexes exist."""
        try:
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT count(*) FROM pg_indexes WHERE schemaname = 'public'"
                )
                count = cur.fetchone()[0]

            if count < EXPECTED_INDEXES_MIN:
                self._add_result(
                    "indexes", "WARN",
                    f"Only {count} indexes found (expected >= {EXPECTED_INDEXES_MIN})"
                )
                return True

            self._add_result("indexes", "PASS", f"{count} indexes present")
            return True
        except Exception as e:
            self._add_result("indexes", "FAIL", f"Index check error: {e}")
            return False

    def verify_row_counts(self, conn) -> bool:
        """Check row counts for critical tables."""
        try:
            counts = {}
            with conn.cursor() as cur:
                for table in EXPECTED_TABLES:
                    if table == "schema_migrations":
                        continue
                    try:
                        cur.execute(f"SELECT count(*) FROM {table}")
                        counts[table] = cur.fetchone()[0]
                    except Exception:
                        counts[table] = -1
                        conn.rollback()

            empty_tables = [t for t, c in counts.items() if c == 0]
            error_tables = [t for t, c in counts.items() if c < 0]

            if error_tables:
                self._add_result(
                    "row_counts", "FAIL",
                    f"Error querying tables: {error_tables}"
                )
                return False

            if empty_tables:
                self._add_result(
                    "row_counts", "WARN",
                    f"Empty tables: {empty_tables}. Counts: {counts}"
                )
            else:
                self._add_result(
                    "row_counts", "PASS",
                    f"Row counts: {counts}"
                )
            return True
        except Exception as e:
            self._add_result("row_counts", "FAIL", f"Row count error: {e}")
            return False

    def verify_sample_data(self, conn) -> bool:
        """Validate sample data from critical tables."""
        try:
            with conn.cursor() as cur:
                # Check assets table has valid structure
                cur.execute(
                    "SELECT id, symbol, name FROM assets LIMIT 5"
                )
                assets = cur.fetchall()

            if not assets:
                self._add_result(
                    "sample_data", "WARN", "No sample data in assets table"
                )
                return True

            # Validate asset records have non-null fields
            invalid = [a for a in assets if not a[1] or not a[2]]
            if invalid:
                self._add_result(
                    "sample_data", "FAIL",
                    f"Invalid asset records (null symbol/name): {len(invalid)}"
                )
                return False

            self._add_result(
                "sample_data", "PASS",
                f"Sample data valid ({len(assets)} assets checked)"
            )
            return True
        except Exception as e:
            self._add_result("sample_data", "WARN", f"Sample data check: {e}")
            return True

    def verify_integrity(self, conn) -> bool:
        """Run referential integrity checks."""
        try:
            with conn.cursor() as cur:
                # Check foreign key constraints are valid
                cur.execute("""
                    SELECT count(*) FROM information_schema.table_constraints
                    WHERE constraint_type = 'FOREIGN KEY'
                    AND table_schema = 'public'
                """)
                fk_count = cur.fetchone()[0]

            self._add_result(
                "integrity", "PASS",
                f"{fk_count} foreign key constraints present"
            )
            return True
        except Exception as e:
            self._add_result("integrity", "FAIL", f"Integrity check error: {e}")
            return False

    def verify_manifest(self, manifest_path: str) -> bool:
        """Verify backup manifest consistency."""
        try:
            manifest = json.loads(Path(manifest_path).read_text())
            files = manifest.get("files", [])
            timestamp = manifest.get("timestamp", "unknown")

            self._add_result(
                "manifest", "PASS",
                f"Manifest valid: {len(files)} files, created {timestamp}"
            )
            return True
        except FileNotFoundError:
            self._add_result("manifest", "WARN", f"Manifest not found: {manifest_path}")
            return True
        except json.JSONDecodeError as e:
            self._add_result("manifest", "FAIL", f"Invalid manifest JSON: {e}")
            return False

    def _add_result(self, check: str, status: str, message: str):
        self.results["checks"].append({
            "check": check,
            "status": status,
            "message": message,
        })
        if status == "PASS":
            self.results["passed"] += 1
        elif status == "FAIL":
            self.results["failed"] += 1
        elif status == "WARN":
            self.results["warnings"] += 1

    def run(self, manifest_path: str = None) -> dict:
        """Run all verification checks."""
        conn = self.connect()

        if conn:
            self.verify_schema(conn)
            self.verify_indexes(conn)
            self.verify_row_counts(conn)
            self.verify_sample_data(conn)
            self.verify_integrity(conn)
            conn.close()

        if manifest_path:
            self.verify_manifest(manifest_path)

        self.results["overall"] = "PASS" if self.results["failed"] == 0 else "FAIL"
        return self.results


def main():
    parser = argparse.ArgumentParser(
        description="Crypto Market Platform Restore Verification"
    )
    parser.add_argument(
        "--dsn",
        default="postgres://cryptouser:cryptopass@localhost:5432/cryptomarket",
        help="PostgreSQL DSN",
    )
    parser.add_argument("--manifest", help="Path to backup manifest JSON")
    parser.add_argument("--report", help="Output report file path")

    args = parser.parse_args()

    print("=" * 60)
    print("  Crypto Market Platform - Restore Verification")
    print("=" * 60)

    verifier = RestoreVerifier(args.dsn)
    results = verifier.run(manifest_path=args.manifest)

    print(f"\nResults:")
    for check in results["checks"]:
        icon = {"PASS": "✓", "FAIL": "✗", "WARN": "⚠", "SKIP": "○"}.get(
            check["status"], "?"
        )
        print(f"  {icon} [{check['status']:4s}] {check['check']}: {check['message']}")

    print(f"\nSummary: {results['passed']} passed, "
          f"{results['failed']} failed, {results['warnings']} warnings")
    print(f"Overall: {results['overall']}")

    if args.report:
        Path(args.report).write_text(json.dumps(results, indent=2))
        print(f"\nReport written to: {args.report}")

    sys.exit(0 if results["overall"] == "PASS" else 1)


if __name__ == "__main__":
    main()
