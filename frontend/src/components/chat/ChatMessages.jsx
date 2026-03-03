// ABOUTME: Scrollable chat message list with sticky-bottom auto-scroll
// ABOUTME: Manages scroll position during streaming and periodic timestamp updates
// ABOUTME: Renders starter prompt chips in empty state and inline errors after messages

import React, { useRef, useEffect, useState, useCallback } from "react";
import { Bot, AlertTriangle } from "lucide-react";
import ChatMessage from "./ChatMessage";

const ERROR_MESSAGES = {
  rate_limit: "Too many requests -- wait a moment and try again",
  timeout: "Response took too long -- try again",
  network: "Connection lost -- check your network and try again",
  server: "Something went wrong -- try again",
};

const STARTER_PROMPTS = [
  {
    label: "Assess current capacity",
    question:
      "Based on the current Diego cell metrics, how much headroom do we have before we need to add more capacity? Consider both memory and CPU utilization.",
  },
  {
    label: "Plan for growth",
    question:
      "If our application workloads grow by 25% over the next quarter, how many additional Diego cells would we need and what hardware should we procure?",
  },
  {
    label: "Review cell sizing",
    question:
      "Are our Diego cells sized optimally? Analyze the current VM specs against the workload distribution and recommend any sizing changes.",
  },
  {
    label: "Check HA readiness",
    question:
      "Evaluate our N-1 redundancy posture. If we lose the largest host in each cluster, do we have enough capacity to absorb the displaced Diego cells?",
  },
];

const InlineError = ({ error, onRetry }) => (
  <div role="alert" className="flex items-start gap-2 px-4 py-3 text-sm">
    <AlertTriangle
      className="w-4 h-4 text-red-400 flex-shrink-0 mt-0.5"
      aria-hidden="true"
    />
    <div>
      <p className="text-red-400">
        {ERROR_MESSAGES[error.type] || ERROR_MESSAGES.server}
      </p>
      <button
        onClick={onRetry}
        className="text-blue-400 hover:text-blue-300 text-xs mt-1 underline"
      >
        Try again
      </button>
    </div>
  </div>
);

const ChatMessages = React.memo(
  ({ messages, isStreaming, error, onRetry, onPromptClick }) => {
    const containerRef = useRef(null);
    const messagesEndRef = useRef(null);
    const shouldAutoScroll = useRef(true);
    const [tick, setTick] = useState(0);

    // Periodic timestamp refresh every 30 seconds
    useEffect(() => {
      const interval = setInterval(() => {
        setTick((prev) => prev + 1);
      }, 30000);
      return () => clearInterval(interval);
    }, []);

    // Auto-scroll to bottom when messages change during streaming
    useEffect(() => {
      if (shouldAutoScroll.current && messagesEndRef.current?.scrollIntoView) {
        messagesEndRef.current.scrollIntoView({ behavior: "smooth" });
      }
    }, [messages, isStreaming]);

    const handleScroll = useCallback(() => {
      const container = containerRef.current;
      if (!container) return;
      const { scrollTop, scrollHeight, clientHeight } = container;
      shouldAutoScroll.current = scrollHeight - scrollTop - clientHeight < 100;
    }, []);

    if (messages.length === 0) {
      return (
        <div className="flex-1 flex flex-col items-center justify-center text-center px-6">
          <div className="w-12 h-12 rounded-xl bg-blue-500/10 border border-blue-500/20 flex items-center justify-center mb-4">
            <Bot className="w-6 h-6 text-blue-400" aria-hidden="true" />
          </div>
          <p className="text-sm text-slate-400 mb-6">
            Ask the AI advisor about your capacity data
          </p>
          <div className="flex flex-wrap justify-center gap-2 max-w-sm">
            {STARTER_PROMPTS.map((prompt, i) => (
              <button
                key={prompt.label}
                onClick={() => onPromptClick(prompt.question)}
                className="px-3 py-1.5 text-xs text-blue-400 border border-blue-500/30 rounded-full hover:bg-blue-500/10 hover:border-blue-500/50 transition-colors"
              >
                {prompt.label}
              </button>
            ))}
          </div>
        </div>
      );
    }

    return (
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto"
      >
        {messages.map((message, index) => (
          <ChatMessage
            key={message.id}
            message={message}
            isStreaming={
              isStreaming &&
              index === messages.length - 1 &&
              message.role === "assistant"
            }
            tick={tick}
          />
        ))}
        {error && <InlineError error={error} onRetry={onRetry} />}
        <div ref={messagesEndRef} />
      </div>
    );
  },
);

ChatMessages.displayName = "ChatMessages";

export default ChatMessages;
