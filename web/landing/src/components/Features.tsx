import React from "react";
import { FEATURES } from "../data/mockData";
import { TrustScoreCards } from "./TrustScoreCards";

interface FeaturesProps {
  readonly className?: string;
}

export const Features: React.FC<FeaturesProps> = ({ className = "" }) => {
  return (
    <section
      className={`py-24 bg-surface-darker/30 border-t border-white/5 ${className}`}
      id="features"
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid lg:grid-cols-2 gap-16 items-center">
          {/* Left: Text + Feature Grid */}
          <div>
            <h2 className="text-3xl md:text-4xl font-bold text-white mb-6">
              Enterprise-grade protection for your codebase
            </h2>
            <p className="text-slate-400 mb-10 text-lg">
              Stop worrying about what your Copilot or ChatGPT generated. We
              validate syntax, security, and business logic.
            </p>
            <div className="grid sm:grid-cols-2 gap-6">
              {FEATURES.map((feature) => (
                <div
                  key={feature.title}
                  className="p-5 bg-surface-dark border border-white/5 rounded-xl hover:border-primary/30 transition-colors"
                >
                  <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center mb-4 text-primary">
                    <span className="material-symbols-outlined">
                      {feature.icon}
                    </span>
                  </div>
                  <h4 className="text-white font-semibold mb-2">
                    {feature.title}
                  </h4>
                  <p className="text-slate-400 text-sm">
                    {feature.description}
                  </p>
                </div>
              ))}
            </div>
          </div>

          {/* Right: Trust Score Visualization */}
          <TrustScoreCards />
        </div>
      </div>
    </section>
  );
};

export default Features;
