// ABOUTME: Tests for chat panel components (ChatPanel, ChatToggle, ChatInput, ChatMessages)
// ABOUTME: Verifies panel lifecycle, conditional toggle, input behavior, message rendering,
// ABOUTME: loading dots, inline errors with retry, reset button, and starter prompts

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ChatPanel from "./ChatPanel";
import ChatToggle from "./ChatToggle";
import ChatInput from "./ChatInput";
import ChatMessages from "./ChatMessages";
import ChatMessage from "./ChatMessage";

// Mock useChatStream hook
vi.mock("../../hooks/useChatStream", () => ({
  useChatStream: vi.fn(() => ({
    messages: [],
    isStreaming: false,
    error: null,
    sendMessage: vi.fn(),
    clearConversation: vi.fn(),
    retryLastMessage: vi.fn(),
  })),
}));

import { useChatStream } from "../../hooks/useChatStream";

// Mock Streamdown to avoid heavy Markdown rendering in unit tests
vi.mock("streamdown", () => ({
  Streamdown: ({ children }) => <div data-testid="streamdown">{children}</div>,
}));

vi.mock("@streamdown/code", () => ({
  code: {},
}));

describe("ChatPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    document.body.style.overflow = "";
  });

  afterEach(() => {
    vi.restoreAllMocks();
    document.body.style.overflow = "";
  });

  it("does not render when never opened", () => {
    render(<ChatPanel isOpen={false} onClose={vi.fn()} />);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders panel when isOpen is true", () => {
    render(<ChatPanel isOpen={true} onClose={vi.fn()} />);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("AI Advisor")).toBeInTheDocument();
  });

  it("calls onClose when backdrop is clicked", () => {
    const onClose = vi.fn();
    render(<ChatPanel isOpen={true} onClose={onClose} />);

    // The backdrop is the first child with aria-hidden
    const backdrop = screen
      .getByRole("dialog")
      .querySelector("[aria-hidden='true']");
    fireEvent.click(backdrop);

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("calls onClose when Escape key is pressed", () => {
    const onClose = vi.fn();
    render(<ChatPanel isOpen={true} onClose={onClose} />);

    fireEvent.keyDown(document, { key: "Escape" });

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("calls onClose when close button is clicked", () => {
    const onClose = vi.fn();
    render(<ChatPanel isOpen={true} onClose={onClose} />);

    fireEvent.click(screen.getByLabelText("Close AI Advisor"));

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("locks body scroll when open", () => {
    render(<ChatPanel isOpen={true} onClose={vi.fn()} />);
    expect(document.body.style.overflow).toBe("hidden");
  });

  it("unlocks body scroll when closed", () => {
    const { rerender } = render(<ChatPanel isOpen={true} onClose={vi.fn()} />);
    expect(document.body.style.overflow).toBe("hidden");

    rerender(<ChatPanel isOpen={false} onClose={vi.fn()} />);
    expect(document.body.style.overflow).toBe("");
  });

  it("displays messages from useChatStream", () => {
    useChatStream.mockReturnValue({
      messages: [
        { id: "msg-1", role: "user", content: "Hello", timestamp: Date.now() },
        {
          id: "msg-2",
          role: "assistant",
          content: "Hi there!",
          timestamp: Date.now(),
        },
      ],
      isStreaming: false,
      error: null,
      sendMessage: vi.fn(),
      clearConversation: vi.fn(),
      retryLastMessage: vi.fn(),
    });

    render(<ChatPanel isOpen={true} onClose={vi.fn()} />);

    expect(screen.getByText("Hello")).toBeInTheDocument();
    expect(screen.getByText("Hi there!")).toBeInTheDocument();
  });

  it("does not render error banner (errors are inline)", () => {
    useChatStream.mockReturnValue({
      messages: [
        { id: "msg-1", role: "user", content: "Hello", timestamp: Date.now() },
      ],
      isStreaming: false,
      error: { message: "Connection lost", type: "network" },
      sendMessage: vi.fn(),
      clearConversation: vi.fn(),
      retryLastMessage: vi.fn(),
    });

    render(<ChatPanel isOpen={true} onClose={vi.fn()} />);

    // The old red bg-red-500/10 error banner should not exist
    const dialog = screen.getByRole("dialog");
    expect(dialog.querySelector(".bg-red-500\\/10")).not.toBeInTheDocument();
  });

  it("renders reset button in header", () => {
    useChatStream.mockReturnValue({
      messages: [],
      isStreaming: false,
      error: null,
      sendMessage: vi.fn(),
      clearConversation: vi.fn(),
      retryLastMessage: vi.fn(),
    });

    render(<ChatPanel isOpen={true} onClose={vi.fn()} />);

    expect(screen.getByLabelText("Reset conversation")).toBeInTheDocument();
  });

  it("calls clearConversation when reset button is clicked", () => {
    const clearConversation = vi.fn();
    useChatStream.mockReturnValue({
      messages: [
        { id: "msg-1", role: "user", content: "Hello", timestamp: Date.now() },
      ],
      isStreaming: false,
      error: null,
      sendMessage: vi.fn(),
      clearConversation,
      retryLastMessage: vi.fn(),
    });

    render(<ChatPanel isOpen={true} onClose={vi.fn()} />);

    fireEvent.click(screen.getByLabelText("Reset conversation"));
    expect(clearConversation).toHaveBeenCalledTimes(1);
  });

  it("calls sendMessage when user types and presses Enter", async () => {
    const sendMessage = vi.fn();
    useChatStream.mockReturnValue({
      messages: [],
      isStreaming: false,
      error: null,
      sendMessage,
      clearConversation: vi.fn(),
      retryLastMessage: vi.fn(),
    });

    render(<ChatPanel isOpen={true} onClose={vi.fn()} />);

    const textarea = screen.getByPlaceholderText(
      "Ask about your capacity data...",
    );
    await userEvent.type(textarea, "How many cells?{enter}");

    expect(sendMessage).toHaveBeenCalledWith("How many cells?");
  });
});

describe("ChatToggle", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders nothing when health returns ai_configured: false", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ ai_configured: false }),
    });

    const { container } = render(
      <ChatToggle isOpen={false} onToggle={vi.fn()} />,
    );

    // Wait for the async effect to complete
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalled();
    });

    expect(container.querySelector("button")).not.toBeInTheDocument();
  });

  it("renders toggle button when health returns ai_configured: true", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ ai_configured: true }),
    });

    render(<ChatToggle isOpen={false} onToggle={vi.fn()} />);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "AI Advisor" }),
      ).toBeInTheDocument();
    });
  });

  it("renders nothing when health fetch fails", async () => {
    global.fetch = vi.fn().mockRejectedValue(new Error("network"));

    const { container } = render(
      <ChatToggle isOpen={false} onToggle={vi.fn()} />,
    );

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalled();
    });

    expect(container.querySelector("button")).not.toBeInTheDocument();
  });

  it("renders nothing when health returns non-OK HTTP status", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 503,
    });

    const { container } = render(
      <ChatToggle isOpen={false} onToggle={vi.fn()} />,
    );

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalled();
    });

    expect(container.querySelector("button")).not.toBeInTheDocument();
  });

  it("applies active styling when isOpen is true", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ ai_configured: true }),
    });

    render(<ChatToggle isOpen={true} onToggle={vi.fn()} />);

    await waitFor(() => {
      const button = screen.getByRole("button", { name: "AI Advisor" });
      expect(button.className).toContain("bg-blue-500/20");
      expect(button).toHaveAttribute("aria-expanded", "true");
    });
  });

  it("calls onToggle when clicked", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ ai_configured: true }),
    });

    const onToggle = vi.fn();
    render(<ChatToggle isOpen={false} onToggle={onToggle} />);

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: "AI Advisor" }),
      ).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "AI Advisor" }));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });
});

describe("ChatInput", () => {
  it("calls onSend with trimmed text on Enter", async () => {
    const onSend = vi.fn();
    render(<ChatInput onSend={onSend} disabled={false} />);

    const textarea = screen.getByPlaceholderText(
      "Ask about your capacity data...",
    );
    await userEvent.type(textarea, "test message{enter}");

    expect(onSend).toHaveBeenCalledWith("test message");
  });

  it("does not call onSend on Shift+Enter", async () => {
    const onSend = vi.fn();
    render(<ChatInput onSend={onSend} disabled={false} />);

    const textarea = screen.getByPlaceholderText(
      "Ask about your capacity data...",
    );
    await userEvent.type(textarea, "line 1{shift>}{enter}{/shift}line 2");

    expect(onSend).not.toHaveBeenCalled();
  });

  it("disables send button when text is empty", () => {
    render(<ChatInput onSend={vi.fn()} disabled={false} />);

    const sendButton = screen.getByLabelText("Send message");
    expect(sendButton).toBeDisabled();
  });

  it("disables send button when disabled prop is true", async () => {
    render(<ChatInput onSend={vi.fn()} disabled={true} />);

    const sendButton = screen.getByLabelText("Send message");
    expect(sendButton).toBeDisabled();
  });

  it("clears text after sending", async () => {
    const onSend = vi.fn();
    render(<ChatInput onSend={onSend} disabled={false} />);

    const textarea = screen.getByPlaceholderText(
      "Ask about your capacity data...",
    );
    await userEvent.type(textarea, "hello{enter}");

    expect(textarea.value).toBe("");
  });
});

describe("ChatMessages", () => {
  it("shows empty state with starter prompts when no messages", () => {
    render(
      <ChatMessages
        messages={[]}
        isStreaming={false}
        error={null}
        onRetry={vi.fn()}
        onPromptClick={vi.fn()}
      />,
    );

    expect(
      screen.getByText("Ask the AI advisor about your capacity data"),
    ).toBeInTheDocument();
  });

  it("renders messages when provided", () => {
    const messages = [
      { id: "msg-1", role: "user", content: "Hello", timestamp: Date.now() },
      {
        id: "msg-2",
        role: "assistant",
        content: "World",
        timestamp: Date.now(),
      },
    ];

    render(
      <ChatMessages
        messages={messages}
        isStreaming={false}
        error={null}
        onRetry={vi.fn()}
        onPromptClick={vi.fn()}
      />,
    );

    expect(screen.getByText("Hello")).toBeInTheDocument();
    expect(screen.getByText("World")).toBeInTheDocument();
  });

  it("renders starter prompt chips when messages is empty", () => {
    render(
      <ChatMessages
        messages={[]}
        isStreaming={false}
        error={null}
        onRetry={vi.fn()}
        onPromptClick={vi.fn()}
      />,
    );

    expect(screen.getByText("Assess current capacity")).toBeInTheDocument();
    expect(screen.getByText("Plan for growth")).toBeInTheDocument();
    expect(screen.getByText("Review cell sizing")).toBeInTheDocument();
    expect(screen.getByText("Check HA readiness")).toBeInTheDocument();
  });

  it("calls onPromptClick with full question when starter prompt is clicked", () => {
    const onPromptClick = vi.fn();
    render(
      <ChatMessages
        messages={[]}
        isStreaming={false}
        error={null}
        onRetry={vi.fn()}
        onPromptClick={onPromptClick}
      />,
    );

    fireEvent.click(screen.getByText("Assess current capacity"));

    expect(onPromptClick).toHaveBeenCalledTimes(1);
    expect(onPromptClick).toHaveBeenCalledWith(
      expect.stringContaining("Diego cell metrics"),
    );
  });

  it("does not show starter prompts when messages is non-empty", () => {
    render(
      <ChatMessages
        messages={[
          {
            id: "msg-1",
            role: "user",
            content: "Hello",
            timestamp: Date.now(),
          },
        ]}
        isStreaming={false}
        error={null}
        onRetry={vi.fn()}
        onPromptClick={vi.fn()}
      />,
    );

    expect(
      screen.queryByText("Assess current capacity"),
    ).not.toBeInTheDocument();
  });

  it("renders inline error with rate_limit message", () => {
    render(
      <ChatMessages
        messages={[
          {
            id: "msg-1",
            role: "user",
            content: "Hello",
            timestamp: Date.now(),
          },
        ]}
        isStreaming={false}
        error={{ message: "rate limited", type: "rate_limit" }}
        onRetry={vi.fn()}
        onPromptClick={vi.fn()}
      />,
    );

    expect(
      screen.getByText("Too many requests -- wait a moment and try again"),
    ).toBeInTheDocument();
  });

  it("renders inline error with network message", () => {
    render(
      <ChatMessages
        messages={[
          {
            id: "msg-1",
            role: "user",
            content: "Hello",
            timestamp: Date.now(),
          },
        ]}
        isStreaming={false}
        error={{ message: "fetch failed", type: "network" }}
        onRetry={vi.fn()}
        onPromptClick={vi.fn()}
      />,
    );

    expect(
      screen.getByText("Connection lost -- check your network and try again"),
    ).toBeInTheDocument();
  });

  it("renders Try again button in inline error", () => {
    render(
      <ChatMessages
        messages={[
          {
            id: "msg-1",
            role: "user",
            content: "Hello",
            timestamp: Date.now(),
          },
        ]}
        isStreaming={false}
        error={{ message: "error", type: "server" }}
        onRetry={vi.fn()}
        onPromptClick={vi.fn()}
      />,
    );

    expect(screen.getByText("Try again")).toBeInTheDocument();
  });

  it("calls onRetry when Try again button is clicked", () => {
    const onRetry = vi.fn();
    render(
      <ChatMessages
        messages={[
          {
            id: "msg-1",
            role: "user",
            content: "Hello",
            timestamp: Date.now(),
          },
        ]}
        isStreaming={false}
        error={{ message: "error", type: "server" }}
        onRetry={onRetry}
        onPromptClick={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByText("Try again"));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});

describe("ChatMessage - Loading dots", () => {
  it("renders loading dots when assistant message has empty content and isStreaming", () => {
    render(
      <ChatMessage
        message={{
          id: "msg-1",
          role: "assistant",
          content: "",
          timestamp: Date.now(),
        }}
        isStreaming={true}
        tick={0}
      />,
    );

    expect(screen.getByLabelText("AI is thinking")).toBeInTheDocument();
  });

  it("does not render loading dots when assistant message has content", () => {
    render(
      <ChatMessage
        message={{
          id: "msg-1",
          role: "assistant",
          content: "Hello there",
          timestamp: Date.now(),
        }}
        isStreaming={true}
        tick={0}
      />,
    );

    expect(screen.queryByLabelText("AI is thinking")).not.toBeInTheDocument();
  });

  it("does not render loading dots when isStreaming is false", () => {
    render(
      <ChatMessage
        message={{
          id: "msg-1",
          role: "assistant",
          content: "",
          timestamp: Date.now(),
        }}
        isStreaming={false}
        tick={0}
      />,
    );

    expect(screen.queryByLabelText("AI is thinking")).not.toBeInTheDocument();
  });
});
