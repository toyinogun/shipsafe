import React from "react";
import { HERO } from "../data/mockData";
import { TerminalDemo } from "./TerminalDemo";

interface HeroProps {
  readonly className?: string;
}

export const Hero: React.FC<HeroProps> = ({ className = "" }) => {
  return (
    <section
      className={`relative pt-32 pb-20 lg:pt-48 lg:pb-32 overflow-hidden ${className}`}
    >
      {/* Background grid */}
      <div className="absolute inset-0 z-0 opacity-20 pointer-events-none grid-bg" />
      {/* Glow orb */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[500px] bg-primary/20 rounded-full blur-[120px] -z-10" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Copy */}
        <div className="text-center max-w-4xl mx-auto mb-16">
          {/* Badge */}
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-surface-dark border border-white/10 mb-6">
            <span className="flex h-2 w-2 relative">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-accent-green opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-accent-green" />
            </span>
            <span className="text-xs font-medium text-slate-300">
              {HERO.badge}
            </span>
          </div>

          <h1 className="text-4xl md:text-6xl lg:text-7xl font-bold text-white tracking-tight mb-6 leading-tight">
            {HERO.headline}{" "}
            <br className="hidden md:block" />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-primary via-blue-400 to-primary/60">
              {HERO.headlineAccent}
            </span>
          </h1>

          <p className="text-lg md:text-xl text-slate-400 max-w-2xl mx-auto mb-10 font-light leading-relaxed">
            {HERO.subheadline}
          </p>

          <div className="flex flex-col sm:flex-row justify-center gap-4">
            <button className="inline-flex items-center justify-center px-8 py-3.5 border border-transparent text-base font-semibold rounded-lg text-white bg-primary hover:bg-blue-600 transition-all shadow-lg shadow-primary/20 hover:shadow-primary/40">
              {HERO.primaryCta}
              <span className="material-symbols-outlined text-sm ml-2">
                arrow_forward
              </span>
            </button>
            <button className="inline-flex items-center justify-center px-8 py-3.5 border border-white/10 bg-surface-dark hover:bg-surface-dark/80 text-base font-medium rounded-lg text-slate-300 hover:text-white transition-all">
              {HERO.secondaryCta}
            </button>
          </div>
        </div>

        {/* Terminal */}
        <TerminalDemo />
      </div>
    </section>
  );
};

export default Hero;
