import { calculateFreshness, getDataAgeSeconds, formatDuration } from "@/lib/freshness";

// Mock config to have predictable thresholds
jest.mock("@/lib/config", () => ({
  config: {
    freshThreshold: 120,
    staleThreshold: 300,
  },
}));

describe("calculateFreshness", () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date("2026-07-20T08:15:00Z"));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("returns 'unknown' for null timestamp", () => {
    expect(calculateFreshness(null)).toBe("unknown");
  });

  it("returns 'unknown' for undefined timestamp", () => {
    expect(calculateFreshness(undefined)).toBe("unknown");
  });

  it("returns 'unknown' for empty string timestamp", () => {
    expect(calculateFreshness("")).toBe("unknown");
  });

  it("returns 'unknown' for invalid timestamp", () => {
    expect(calculateFreshness("not-a-date")).toBe("unknown");
  });

  it("returns 'fresh' for recent data (within 120s)", () => {
    const recent = new Date("2026-07-20T08:14:00Z").toISOString(); // 60s ago
    expect(calculateFreshness(recent)).toBe("fresh");
  });

  it("returns 'fresh' for data exactly at threshold", () => {
    const atThreshold = new Date("2026-07-20T08:13:00Z").toISOString(); // 120s ago
    expect(calculateFreshness(atThreshold)).toBe("fresh");
  });

  it("returns 'delayed' for data between fresh and stale thresholds", () => {
    const delayed = new Date("2026-07-20T08:11:00Z").toISOString(); // 240s ago
    expect(calculateFreshness(delayed)).toBe("delayed");
  });

  it("returns 'stale' for old data (beyond 300s)", () => {
    const stale = new Date("2026-07-20T08:09:00Z").toISOString(); // 360s ago
    expect(calculateFreshness(stale)).toBe("stale");
  });

  it("supports custom thresholds", () => {
    const timestamp = new Date("2026-07-20T08:14:00Z").toISOString(); // 60s ago
    // With a 30s fresh threshold, 60s old data should be delayed
    expect(calculateFreshness(timestamp, 30, 300)).toBe("delayed");
  });
});

describe("getDataAgeSeconds", () => {
  beforeEach(() => {
    jest.useFakeTimers();
    jest.setSystemTime(new Date("2026-07-20T08:15:00Z"));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it("returns null for null timestamp", () => {
    expect(getDataAgeSeconds(null)).toBeNull();
  });

  it("returns null for invalid timestamp", () => {
    expect(getDataAgeSeconds("invalid")).toBeNull();
  });

  it("returns correct age in seconds", () => {
    const timestamp = new Date("2026-07-20T08:14:00Z").toISOString(); // 60s ago
    expect(getDataAgeSeconds(timestamp)).toBe(60);
  });
});

describe("formatDuration", () => {
  it("formats seconds", () => {
    expect(formatDuration(30)).toBe("30s ago");
  });

  it("formats minutes", () => {
    expect(formatDuration(120)).toBe("2m ago");
  });

  it("formats hours", () => {
    expect(formatDuration(7200)).toBe("2h ago");
  });

  it("formats boundary at 60s", () => {
    expect(formatDuration(60)).toBe("1m ago");
  });

  it("formats boundary at 3600s", () => {
    expect(formatDuration(3600)).toBe("1h ago");
  });
});
