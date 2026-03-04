// ABOUTME: Renders a single chat message with icon, content, and relative timestamp
// ABOUTME: Uses Streamdown for streaming Markdown in assistant messages; memoized to prevent re-render storms
// ABOUTME: Shows pulsing dots indicator when assistant message is streaming with empty content
// ABOUTME: Displays floating action bar with copy-to-clipboard and feedback buttons on completed assistant messages

import React, {
  useMemo,
  useState,
  useCallback,
  useRef,
  useEffect,
} from "react";
import { User, Bot, Copy, Check, ThumbsUp, ThumbsDown } from "lucide-react";
import { Streamdown } from "streamdown";
import { code } from "@streamdown/code";
import "streamdown/styles.css";
import { formatRelativeTime } from "../../utils/formatRelativeTime";
import { stripMarkdown } from "../../utils/stripMarkdown";

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

const LoadingDots = () => (
  <div className="flex items-center gap-1 py-1" aria-label="AI is thinking">
    {[0, 1, 2].map((i) => (
      <div
        key={i}
        className="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse"
        style={{ animationDelay: `${i * 200}ms` }}
      />
    ))}
  </div>
);

function CopyButton({ content }) {
  const [status, setStatus] = useState("idle"); // idle | copied | failed
  const timerRef = useRef(null);

  useEffect(() => () => clearTimeout(timerRef.current), []);

  const handleCopy = useCallback(async () => {
    clearTimeout(timerRef.current);
    try {
      await navigator.clipboard.writeText(stripMarkdown(content));
      setStatus("copied");
    } catch (err) {
      console.warn("Clipboard write failed:", err);
      setStatus("failed");
    }
    timerRef.current = setTimeout(() => setStatus("idle"), 2000);
  }, [content]);

  const label =
    status === "copied"
      ? "Copied"
      : status === "failed"
        ? "Copy failed"
        : "Copy to clipboard";

  return (
    <button
      onClick={handleCopy}
      className="p-1 rounded text-slate-400 hover:text-slate-200 transition-colors"
      aria-label={label}
    >
      {status === "copied" ? (
        <Check className="w-3.5 h-3.5 text-green-400" />
      ) : status === "failed" ? (
        <Copy className="w-3.5 h-3.5 text-red-400" />
      ) : (
        <Copy className="w-3.5 h-3.5" />
      )}
    </button>
  );
}

function FeedbackButtons({ rating, onRate }) {
  return (
    <>
      <button
        onClick={() => onRate("up")}
        className={`p-1 rounded transition-colors ${
          rating === "up"
            ? "text-green-400"
            : "text-slate-400 hover:text-slate-200"
        }`}
        aria-label={
          rating === "up" ? "Remove positive feedback" : "Good response"
        }
        aria-pressed={rating === "up"}
      >
        <ThumbsUp className="w-3.5 h-3.5" />
      </button>
      <button
        onClick={() => onRate("down")}
        className={`p-1 rounded transition-colors ${
          rating === "down"
            ? "text-red-400"
            : "text-slate-400 hover:text-slate-200"
        }`}
        aria-label={
          rating === "down" ? "Remove negative feedback" : "Poor response"
        }
        aria-pressed={rating === "down"}
      >
        <ThumbsDown className="w-3.5 h-3.5" />
      </button>
    </>
  );
}

function MessageContent({ message, isStreaming, plugins }) {
  if (message.role !== "assistant") {
    return (
      <p className="text-sm text-slate-200 whitespace-pre-wrap">
        {message.content}
      </p>
    );
  }

  if (message.content === "" && isStreaming) {
    return <LoadingDots />;
  }

  return (
    <div className="streamdown-content">
      <Streamdown
        plugins={plugins}
        components={markdownComponents}
        isAnimating={isStreaming}
      >
        {message.content}
      </Streamdown>
    </div>
  );
}

const ChatMessage = React.memo(
  ({ message, isStreaming, tick: _tick, feedbackRating, onFeedback }) => {
    const isAssistant = message.role === "assistant";
    const plugins = useMemo(() => ({ code }), []);
    const showActionBar = isAssistant && message.content && !isStreaming;

    return (
      <div
        className={`group relative flex gap-3 px-4 py-3 ${isAssistant ? "bg-slate-800/30" : ""}`}
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
          <MessageContent
            message={message}
            isStreaming={isStreaming}
            plugins={plugins}
          />
          <span className="text-xs text-slate-500 mt-1 block">
            {formatRelativeTime(message.timestamp)}
          </span>
        </div>
        {showActionBar && (
          <div
            data-testid="action-bar"
            className="absolute top-2 right-2 md:opacity-0 md:pointer-events-none md:group-hover:opacity-100 md:group-hover:pointer-events-auto md:group-focus-within:opacity-100 md:group-focus-within:pointer-events-auto transition-opacity flex items-center gap-0.5 bg-slate-800 border border-slate-600 rounded-md px-1 py-0.5"
          >
            <CopyButton content={message.content} />
            <FeedbackButtons
              rating={feedbackRating}
              onRate={(rating) => onFeedback?.(rating)}
            />
          </div>
        )}
      </div>
    );
  },
);

ChatMessage.displayName = "ChatMessage";

export default ChatMessage;
