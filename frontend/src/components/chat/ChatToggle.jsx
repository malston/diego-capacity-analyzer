// ABOUTME: Toggle button for the AI advisor chat panel
// ABOUTME: Conditionally renders based on ai_configured from health endpoint

import { useState, useEffect } from "react";
import { MessageSquare } from "lucide-react";

const ChatToggle = ({ isOpen, onToggle }) => {
  const [aiConfigured, setAiConfigured] = useState(false);

  useEffect(() => {
    const checkAI = async () => {
      try {
        const apiURL = import.meta.env.VITE_API_URL || "";
        const response = await fetch(`${apiURL}/api/v1/health`, {
          credentials: "include",
        });
        if (!response.ok) {
          setAiConfigured(false);
          return;
        }
        const data = await response.json();
        setAiConfigured(data.ai_configured === true);
      } catch {
        setAiConfigured(false);
      }
    };
    checkAI();
  }, []);

  if (!aiConfigured) {
    return null;
  }

  return (
    <button
      onClick={onToggle}
      className={`p-2 rounded-lg transition-all ${
        isOpen
          ? "bg-blue-500/20 text-blue-400 border border-blue-500/30"
          : "bg-slate-800/50 text-slate-400 border border-slate-700 hover:text-slate-300 hover:border-slate-600"
      }`}
      aria-label="AI Advisor"
      aria-expanded={isOpen}
    >
      <MessageSquare className="w-5 h-5" aria-hidden="true" />
    </button>
  );
};

export default ChatToggle;
