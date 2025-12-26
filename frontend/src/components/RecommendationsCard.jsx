// ABOUTME: Upgrade recommendations display component
// ABOUTME: Shows actionable recommendations with priority ordering

import { Lightbulb, Plus, ArrowUp, Server, Zap, Settings } from 'lucide-react';

// Get icon for recommendation type
const getTypeIcon = (type) => {
  switch (type) {
    case 'scale-out':
      return Plus;
    case 'scale-up':
      return ArrowUp;
    case 'infrastructure':
      return Server;
    case 'optimization':
      return Zap;
    default:
      return Settings;
  }
};

// Get impact color
const getImpactColor = (impact) => {
  switch (impact) {
    case 'high':
      return { text: 'text-emerald-400', bg: 'bg-emerald-500/20' };
    case 'medium':
      return { text: 'text-amber-400', bg: 'bg-amber-500/20' };
    case 'low':
      return { text: 'text-gray-400', bg: 'bg-gray-500/20' };
    default:
      return { text: 'text-gray-400', bg: 'bg-gray-500/20' };
  }
};

// Get type badge color
const getTypeBadgeColor = (type) => {
  switch (type) {
    case 'scale-out':
      return 'bg-cyan-500/20 text-cyan-400';
    case 'scale-up':
      return 'bg-purple-500/20 text-purple-400';
    case 'infrastructure':
      return 'bg-blue-500/20 text-blue-400';
    case 'optimization':
      return 'bg-emerald-500/20 text-emerald-400';
    default:
      return 'bg-gray-500/20 text-gray-400';
  }
};

const RecommendationsCard = ({ recommendations = [] }) => {
  // Sort by priority (lowest number = highest priority)
  const sortedRecommendations = [...recommendations].sort(
    (a, b) => a.priority - b.priority
  );

  if (recommendations.length === 0) {
    return (
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
        <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
          <Lightbulb size={18} className="text-amber-400" />
          Upgrade Recommendations
        </h3>
        <p className="text-gray-400 text-sm">No recommendations available</p>
      </div>
    );
  }

  return (
    <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl p-6 border border-slate-700/50">
      <h3 className="text-lg font-semibold mb-4 text-gray-200 flex items-center gap-2">
        <Lightbulb size={18} className="text-amber-400" />
        Upgrade Recommendations
      </h3>

      <div className="space-y-3">
        {sortedRecommendations.map((rec) => {
          const TypeIcon = getTypeIcon(rec.type);
          // Use impact_level for badge coloring (falls back to impact for backwards compat)
          const impactLevel = rec.impact_level || rec.impact;
          const impactColors = getImpactColor(impactLevel);
          const typeBadgeColor = getTypeBadgeColor(rec.type);

          return (
            <div
              key={rec.id}
              data-recommendation={rec.id}
              className="bg-slate-700/30 rounded-lg p-4 border border-slate-600/30"
            >
              <div className="flex items-start gap-3">
                {/* Priority Number */}
                <div className="w-8 h-8 rounded-full bg-cyan-500/20 flex items-center justify-center flex-shrink-0">
                  <span className="text-cyan-400 font-bold text-sm">{rec.priority}</span>
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                  {/* Title Row */}
                  <div className="flex items-center gap-2 flex-wrap">
                    <TypeIcon size={16} className="text-cyan-400 flex-shrink-0" />
                    <h4 className="font-medium text-gray-200">{rec.title}</h4>

                    {/* Type Badge */}
                    <span
                      className={`text-xs px-2 py-0.5 rounded ${typeBadgeColor}`}
                    >
                      {rec.type}
                    </span>

                    {/* Impact Level Badge */}
                    {impactLevel && (
                      <span
                        className={`text-xs px-2 py-0.5 rounded ${impactColors.bg} ${impactColors.text}`}
                      >
                        {impactLevel} impact
                      </span>
                    )}
                  </div>

                  {/* Description */}
                  {rec.description && (
                    <p className="text-sm text-gray-400 mt-1">{rec.description}</p>
                  )}

                  {/* Impact Description (detailed text from backend) */}
                  {rec.impact && rec.impact_level && (
                    <p className="text-xs text-gray-500 mt-1 italic">{rec.impact}</p>
                  )}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default RecommendationsCard;
