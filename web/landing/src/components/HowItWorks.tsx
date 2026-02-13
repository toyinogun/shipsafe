import React from "react";
import { STEPS } from "../data/mockData";

interface HowItWorksProps {
  readonly className?: string;
}

export const HowItWorks: React.FC<HowItWorksProps> = ({ className = "" }) => {
  return (
    <section className={`py-24 relative ${className}`} id="how-it-works">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center mb-16">
          <h2 className="text-3xl md:text-4xl font-bold text-white mb-4">
            Verification Flow
          </h2>
          <p className="text-slate-400 max-w-2xl mx-auto">
            Seamlessly integrates into your existing workflow without disrupting
            velocity.
          </p>
        </div>

        <div className="relative grid md:grid-cols-3 gap-8">
          {/* Connector line (desktop) */}
          <div className="hidden md:block absolute top-12 left-[16%] right-[16%] h-px bg-gradient-to-r from-transparent via-primary/50 to-transparent border-t border-dashed border-slate-700 -z-10" />

          {STEPS.map((step) => (
            <div key={step.title} className="relative group">
              <div className="w-24 h-24 mx-auto bg-surface-dark border border-slate-700 rounded-2xl flex items-center justify-center mb-6 shadow-lg group-hover:border-primary/50 group-hover:shadow-primary/20 transition-all">
                <span className="material-symbols-outlined text-4xl text-slate-300 group-hover:text-primary transition-colors">
                  {step.icon}
                </span>
              </div>
              <div className="text-center">
                <h3 className="text-xl font-semibold text-white mb-2">
                  {step.title}
                </h3>
                <p className="text-slate-400 text-sm leading-relaxed">
                  {step.description}
                </p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default HowItWorks;
