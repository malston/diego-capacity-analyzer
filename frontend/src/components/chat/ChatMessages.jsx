// ABOUTME: Scrollable chat message list with sticky-bottom auto-scroll
// ABOUTME: Manages scroll position during streaming and periodic timestamp updates

import React, { useRef, useEffect, useState, useCallback } from "react";
import { Bot } from "lucide-react";
import ChatMessage from "./ChatMessage";

const ChatMessages = React.memo(({ messages, isStreaming }) => {
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
        <p className="text-sm text-slate-400">
          Ask the AI advisor about your capacity data
        </p>
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
          key={index}
          message={message}
          isStreaming={
            isStreaming &&
            index === messages.length - 1 &&
            message.role === "assistant"
          }
          tick={tick}
        />
      ))}
      <div ref={messagesEndRef} />
    </div>
  );
});

ChatMessages.displayName = "ChatMessages";

export default ChatMessages;
