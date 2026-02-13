import React from "react";
import { TRUST_SCORE_CARDS } from "../data/mockData";

interface TrustScoreCardsProps {
  readonly className?: string;
}

const statusStyles = {
  red: {
    border: "border-red-500/30",
    iconColor: "text-red-500",
    badgeBg: "bg-red-500/20",
    badgeText: "text-red-400",
    barColor: "bg-red-500",
    wrapper: "translate-x-4 opacity-60 scale-95",
  },
  yellow: {
    border: "border-yellow-500/30",
    iconColor: "text-yellow-500",
    badgeBg: "bg-yellow-500/20",
    badgeText: "text-yellow-400",
    barColor: "bg-yellow-500",
    wrapper: "-translate-x-4 opacity-80 scale-[0.98] z-10",
  },
  green: {
    border: "border-green-500/50",
    iconColor: "text-green-500",
    badgeBg: "bg-green-500/20",
    badgeText: "text-green-400",
    barColor: "bg-green-500",
    wrapper: "scale-100 z-20",
  },
} as const;

export const TrustScoreCards: React.FC<TrustScoreCardsProps> = ({
  className = "",
}) => {
  return (
    <div className={`relative ${className}`}>
      {/* Background glow */}
      <div className="absolute inset-0 bg-primary/20 blur-[100px] rounded-full -z-10 opacity-30" />

      <div className="flex flex-col gap-4">
        {TRUST_SCORE_CARDS.map((card) => {
          const style = statusStyles[card.status];
          const isGreen = card.status === "green";

          return (
            <div
              key={card.pr}
              className={`bg-surface-dark ${style.border} rounded-xl ${isGreen ? "p-6" : "p-4"} transform ${style.wrapper} shadow-lg ${isGreen ? "shadow-2xl shadow-green-900/20" : ""}`}
            >
              <div
                className={`flex items-center justify-between ${isGreen ? "mb-4" : "mb-2"}`}
              >
                <div className="flex items-center gap-2">
                  {isGreen ? (
                    <div className="bg-green-500/20 p-2 rounded-lg">
                      <span
                        className={`material-symbols-outlined ${style.iconColor}`}
                      >
                        {card.icon}
                      </span>
                    </div>
                  ) : (
                    <span
                      className={`material-symbols-outlined ${style.iconColor}`}
                    >
                      {card.icon}
                    </span>
                  )}
                  <div>
                    <span
                      className={`text-slate-300 font-mono text-sm ${isGreen ? "block text-white font-semibold font-[Inter]" : ""}`}
                    >
                      {isGreen
                        ? `${card.pr} - ${card.repo}`
                        : `${card.pr} - ${card.repo}`}
                    </span>
                    {isGreen && "author" in card && (
                      <span className="text-slate-400 text-xs">
                        Authored by {card.author} &bull; Verified by ShipSafe
                      </span>
                    )}
                  </div>
                </div>

                {isGreen ? (
                  <div className="text-right">
                    <div className="text-2xl font-bold text-white">
                      {card.score}
                      <small className="text-slate-500 text-base">/100</small>
                    </div>
                    <div className="text-green-400 text-xs font-bold uppercase tracking-wide">
                      {card.label}
                    </div>
                  </div>
                ) : (
                  <span
                    className={`${style.badgeBg} ${style.badgeText} text-xs px-2 py-1 rounded-sm font-bold`}
                  >
                    {card.label} {card.score}/100
                  </span>
                )}
              </div>

              {/* Progress bars */}
              {isGreen && "metrics" in card ? (
                <div className="space-y-3">
                  {card.metrics.map((metric) => (
                    <React.Fragment key={metric.label}>
                      <div className="flex justify-between text-xs text-slate-400">
                        <span>{metric.label}</span>
                        <span className="text-white">{metric.value}%</span>
                      </div>
                      <div className="h-1.5 w-full bg-slate-700 rounded-full overflow-hidden">
                        <div
                          className={`h-full ${style.barColor}`}
                          style={{ width: `${metric.value}%` }}
                        />
                      </div>
                    </React.Fragment>
                  ))}
                  <button className="mt-6 w-full py-2 bg-green-600 hover:bg-green-500 text-white text-sm font-medium rounded-lg transition-colors shadow-lg shadow-green-900/20">
                    Approve Merge Request
                  </button>
                </div>
              ) : (
                <div className="h-1.5 w-full bg-slate-700 rounded-full overflow-hidden">
                  <div
                    className={`h-full ${style.barColor}`}
                    style={{ width: `${card.score}%` }}
                  />
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default TrustScoreCards;
