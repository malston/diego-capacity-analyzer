// ABOUTME: Scrollable chat message list with sticky-bottom auto-scroll, data source banner
// ABOUTME: Manages scroll position during streaming and periodic timestamp updates
// ABOUTME: Renders adaptive starter prompts in empty state based on available data sources
// ABOUTME: Exports DataSourceBanner for amber info banner when BOSH/vSphere unavailable

import React, { useRef, useEffect, useState, useCallback } from "react";
import { Bot, AlertTriangle, Info } from "lucide-react";
import ChatMessage from "./ChatMessage";

const ERROR_MESSAGES = {
  rate_limit: "Too many requests -- wait a moment and try again",
  timeout: "Response took too long -- try again",
  network: "Connection lost -- check your network and try again",
  server: "Something went wrong -- try again",
};

const ALL_PROMPTS = [
  {
    label: "Assess current capacity",
    question:
      "Based on the current Diego cell metrics, how much headroom do we have before we need to add more capacity? Consider both memory and CPU utilization.",
    requires: ["bosh"],
  },
  {
    label: "Plan for growth",
    question:
      "If our application workloads grow by 25% over the next quarter, how many additional Diego cells would we need and what hardware should we procure?",
    requires: ["bosh"],
  },
  {
    label: "Review cell sizing",
    question:
      "Are our Diego cells sized optimally? Analyze the current VM specs against the workload distribution and recommend any sizing changes.",
    requires: ["bosh"],
  },
  {
    label: "Check HA readiness",
    question:
      "Evaluate our N-1 redundancy posture. If we lose the largest host in each cluster, do we have enough capacity to absorb the displaced Diego cells?",
    requires: ["bosh", "vsphere"],
  },
  {
    label: "Review app distribution",
    question:
      "Analyze how applications are distributed across isolation segments. Are there any segments that are over or under-utilized?",
    requires: ["cf"],
  },
  {
    label: "Analyze memory allocation",
    question:
      "Review the memory allocation patterns across our applications. Are there apps that are over-provisioned or under-provisioned relative to their actual usage?",
    requires: ["cf"],
  },
  {
    label: "Check isolation segments",
    question:
      "How are our isolation segments configured and which applications run in each? Are there any segmentation improvements we should consider?",
    requires: ["cf"],
  },
  {
    label: "Assess app density",
    question:
      "What is the current application density? Are there any apps consuming significantly more resources than others?",
    requires: ["cf"],
  },
];

function getAvailablePrompts(dataSources, maxPrompts = 4) {
  if (dataSources === null || dataSources === undefined) {
    return ALL_PROMPTS.slice(0, maxPrompts);
  }

  const available = new Set(["cf"]);
  if (dataSources.bosh) available.add("bosh");
  if (dataSources.vsphere) available.add("vsphere");

  return ALL_PROMPTS.filter((p) =>
    p.requires.every((req) => available.has(req)),
  ).slice(0, maxPrompts);
}

export const DataSourceBanner = ({ dataSources }) => {
  if (!dataSources) return null;

  const missing = [];
  if (!dataSources.bosh) missing.push("BOSH");
  if (!dataSources.vsphere) missing.push("vSphere");

  if (missing.length === 0) return null;

  return (
    <div
      role="status"
      className="px-4 py-2 bg-amber-500/10 border-b border-amber-500/20 text-amber-300 text-xs flex items-center gap-2 flex-shrink-0"
    >
      <Info className="w-3.5 h-3.5 flex-shrink-0" aria-hidden="true" />
      <span>{missing.join(" and ")} data unavailable</span>
    </div>
  );
};

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
  ({ messages, isStreaming, error, onRetry, onPromptClick, dataSources }) => {
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
            {getAvailablePrompts(dataSources).map((prompt) => (
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
