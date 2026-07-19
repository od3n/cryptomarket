import { render, screen } from "@testing-library/react";
import { FreshnessBadge } from "@/components/FreshnessBadge";

// Mock config to have predictable thresholds
jest.mock("@/lib/config", () => ({
  config: {
    freshThreshold: 120,
    staleThreshold: 300,
  },
}));

describe("FreshnessBadge", () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date("2026-07-20T08:15:00Z"));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("renders 'Fresh' for recent timestamp", () => {
    const recent = new Date("2026-07-20T08:14:00Z").toISOString(); // 60s ago
    render(<FreshnessBadge timestamp={recent} />);
    expect(screen.getByText("Fresh")).toBeInTheDocument();
  });

  it("renders 'Delayed' for moderately old timestamp", () => {
    const delayed = new Date("2026-07-20T08:11:00Z").toISOString(); // 240s ago
    render(<FreshnessBadge timestamp={delayed} />);
    expect(screen.getByText("Delayed")).toBeInTheDocument();
  });

  it("renders 'Stale' for old timestamp", () => {
    const stale = new Date("2026-07-20T08:09:00Z").toISOString(); // 360s ago
    render(<FreshnessBadge timestamp={stale} />);
    expect(screen.getByText("Stale")).toBeInTheDocument();
  });

  it("renders 'Unknown' for null timestamp", () => {
    render(<FreshnessBadge timestamp={null} />);
    expect(screen.getByText("Unknown")).toBeInTheDocument();
  });

  it("shows age when showAge is true", () => {
    const recent = new Date("2026-07-20T08:14:30Z").toISOString(); // 30s ago
    render(<FreshnessBadge timestamp={recent} showAge={true} />);
    expect(screen.getByText("(30s ago)")).toBeInTheDocument();
  });

  it("does not show age by default", () => {
    const recent = new Date("2026-07-20T08:14:00Z").toISOString();
    render(<FreshnessBadge timestamp={recent} />);
    expect(screen.queryByText(/\d+s ago/)).not.toBeInTheDocument();
  });
});
