import { render, screen } from "@testing-library/react";
import { ConnectionStatus } from "@/components/ConnectionStatus";
import { ConnectionState } from "@/types/market";

describe("ConnectionStatus", () => {
  const states: { state: ConnectionState; label: string }[] = [
    { state: "connecting", label: "Connecting" },
    { state: "live", label: "Live" },
    { state: "reconnecting", label: "Reconnecting" },
    { state: "disconnected", label: "Disconnected" },
    { state: "degraded", label: "Degraded" },
  ];

  it.each(states)("renders '$label' for state '$state'", ({ state, label }) => {
    render(<ConnectionStatus state={state} />);
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it("has correct aria-label", () => {
    render(<ConnectionStatus state="live" />);
    expect(screen.getByRole("status")).toHaveAttribute(
      "aria-label",
      "Connection status: Live"
    );
  });

  it("shows pulse animation for connecting state", () => {
    const { container } = render(<ConnectionStatus state="connecting" />);
    expect(container.querySelector(".animate-ping")).toBeInTheDocument();
  });

  it("does not show pulse animation for live state", () => {
    const { container } = render(<ConnectionStatus state="live" />);
    expect(container.querySelector(".animate-ping")).not.toBeInTheDocument();
  });
});
