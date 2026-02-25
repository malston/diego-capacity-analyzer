// ABOUTME: Renders a single chat message with icon, content, and relative timestamp
// ABOUTME: Uses Streamdown for streaming Markdown in assistant messages; memoized to prevent re-render storms

import React, { useMemo } from "react";
import { User, Bot } from "lucide-react";
import { Streamdown } from "streamdown";
import { code } from "@streamdown/code";
import "streamdown/styles.css";
import { formatRelativeTime } from "../../utils/formatRelativeTime";

const markdownComponents = {
  h1: ({ children, ...props }) => (
    <h1 className="text-lg font-bold text-white mt-4 mb-2" {...props}>
      {children}
    </h1>
  ),
  h2: ({ children, ...props }) => (
    <h2 className="text-base font-semibold text-white mt-3 mb-1.5" {...props}>
      {children}
    </h2>
  ),
  h3: ({ children, ...props }) => (
    <h3 className="text-sm font-semibold text-white mt-2 mb-1" {...props}>
      {children}
    </h3>
  ),
  p: ({ children, ...props }) => (
    <p className="text-sm text-slate-200 leading-relaxed mb-2" {...props}>
      {children}
    </p>
  ),
  strong: ({ children }) => (
    <strong className="font-semibold text-white">{children}</strong>
  ),
  em: ({ children }) => <em className="text-slate-300 italic">{children}</em>,
  ul: ({ children }) => (
    <ul className="list-disc list-inside space-y-1 text-sm text-slate-200 mb-2">
      {children}
    </ul>
  ),
  ol: ({ children }) => (
    <ol className="list-decimal list-inside space-y-1 text-sm text-slate-200 mb-2">
      {children}
    </ol>
  ),
  li: ({ children }) => <li className="text-sm text-slate-200">{children}</li>,
  table: ({ children }) => (
    <div className="overflow-x-auto my-2">
      <table className="min-w-full text-xs border-collapse border border-slate-700">
        {children}
      </table>
    </div>
  ),
  thead: ({ children }) => <thead>{children}</thead>,
  tbody: ({ children }) => <tbody>{children}</tbody>,
  tr: ({ children }) => (
    <tr className="border-b border-slate-700">{children}</tr>
  ),
  th: ({ children }) => (
    <th className="border border-slate-700 px-2 py-1 bg-slate-800 text-slate-300 text-left">
      {children}
    </th>
  ),
  td: ({ children }) => (
    <td className="border border-slate-700 px-2 py-1 text-slate-300">
      {children}
    </td>
  ),
  pre: ({ children, ...props }) => (
    <pre
      className="bg-slate-900 border border-slate-700 rounded-lg p-3 overflow-x-auto my-2 text-xs"
      {...props}
    >
      {children}
    </pre>
  ),
  code: ({ children, className, ...props }) => {
    const isInline = !className?.includes("language-");
    return isInline ? (
      <code
        className="bg-slate-800 px-1.5 py-0.5 rounded text-xs text-blue-300"
        {...props}
      >
        {children}
      </code>
    ) : (
      <code className={className} {...props}>
        {children}
      </code>
    );
  },
  blockquote: ({ children }) => (
    <blockquote className="border-l-2 border-blue-500 pl-3 my-2 text-slate-400 italic">
      {children}
    </blockquote>
  ),
  a: ({ children, href, ...props }) => (
    <a
      href={href}
      className="text-blue-400 hover:text-blue-300 underline"
      target="_blank"
      rel="noopener noreferrer"
      {...props}
    >
      {children}
    </a>
  ),
  hr: () => <hr className="border-slate-700 my-3" />,
};

const ChatMessage = React.memo(({ message, isStreaming, tick: _tick }) => {
  const isAssistant = message.role === "assistant";
  const plugins = useMemo(() => ({ code }), []);

  return (
    <div
      className={`flex gap-3 px-4 py-3 ${isAssistant ? "bg-slate-800/30" : ""}`}
    >
      <div className="flex-shrink-0 mt-0.5">
        {isAssistant ? (
          <div className="w-7 h-7 rounded-md bg-blue-500/20 border border-blue-500/30 flex items-center justify-center">
            <Bot className="w-4 h-4 text-blue-400" aria-hidden="true" />
          </div>
        ) : (
          <div className="w-7 h-7 rounded-md bg-slate-700/50 border border-slate-600 flex items-center justify-center">
            <User className="w-4 h-4 text-slate-400" aria-hidden="true" />
          </div>
        )}
      </div>
      <div className="flex-1 min-w-0">
        {isAssistant ? (
          <div className="streamdown-content">
            <Streamdown
              plugins={plugins}
              components={markdownComponents}
              isAnimating={isStreaming}
            >
              {message.content}
            </Streamdown>
          </div>
        ) : (
          <p className="text-sm text-slate-200 whitespace-pre-wrap">
            {message.content}
          </p>
        )}
        <span className="text-xs text-slate-500 mt-1 block">
          {formatRelativeTime(message.timestamp)}
        </span>
      </div>
    </div>
  );
});

ChatMessage.displayName = "ChatMessage";

export default ChatMessage;
