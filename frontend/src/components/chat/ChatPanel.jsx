// ABOUTME: Overlay chat panel with slide-in animation, backdrop, and responsive layout
// ABOUTME: Manages panel lifecycle, body scroll lock, and abort on close

import { useEffect, useCallback } from "react";
import { X } from "lucide-react";
import { useChatStream } from "../../hooks/useChatStream";
import ChatMessages from "./ChatMessages";
import ChatInput from "./ChatInput";

const ChatPanel = ({ isOpen, onClose }) => {
  const { messages, isStreaming, sendMessage } = useChatStream();

  // Body scroll lock when panel is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [isOpen]);

  // Escape key handler
  const handleEscape = useCallback(
    (e) => {
      if (e.key === "Escape") {
        onClose();
      }
    },
    [onClose],
  );

  useEffect(() => {
    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
    }
    return () => {
      document.removeEventListener("keydown", handleEscape);
    };
  }, [isOpen, handleEscape]);

  return (
    <div
      className={`fixed inset-0 z-40 ${isOpen ? "" : "pointer-events-none"}`}
      role="dialog"
      aria-label="AI Advisor"
      aria-hidden={!isOpen}
    >
      {/* Backdrop */}
      <div
        className={`absolute inset-0 bg-black/50 transition-opacity duration-300 ${
          isOpen ? "opacity-100" : "opacity-0"
        }`}
        onClick={onClose}
        aria-hidden="true"
      />
      {/* Panel */}
      <div
        className={`absolute top-0 right-0 h-full w-full sm:w-[440px] bg-slate-900 border-l border-slate-700
          transform transition-transform duration-300 ease-out flex flex-col
          ${isOpen ? "translate-x-0" : "translate-x-full"}`}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-slate-700 flex-shrink-0">
          <h2 className="text-sm font-semibold text-white">AI Advisor</h2>
          <button
            onClick={onClose}
            className="p-1.5 rounded-md text-slate-400 hover:text-slate-300 hover:bg-slate-800 transition-colors"
            aria-label="Close AI Advisor"
          >
            <X className="w-4 h-4" aria-hidden="true" />
          </button>
        </div>

        {/* Messages */}
        <ChatMessages messages={messages} isStreaming={isStreaming} />

        {/* Input */}
        <ChatInput onSend={sendMessage} disabled={isStreaming} />
      </div>
    </div>
  );
};

export default ChatPanel;
