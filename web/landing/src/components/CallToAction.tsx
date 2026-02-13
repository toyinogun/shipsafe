import React from "react";
import { CTA } from "../data/mockData";

interface CallToActionProps {
  readonly className?: string;
}

export const CallToAction: React.FC<CallToActionProps> = ({
  className = "",
}) => {
  return (
    <section className={`py-24 relative overflow-hidden ${className}`}>
      <div className="absolute inset-0 bg-primary/5" />
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10 text-center">
        <h2 className="text-4xl md:text-5xl font-bold text-white mb-6">
          {CTA.headline}
        </h2>
        <p className="text-xl text-slate-400 mb-10">{CTA.subheadline}</p>
        <div className="flex flex-col sm:flex-row justify-center gap-4">
          <button className="inline-flex items-center justify-center px-8 py-4 border border-transparent text-lg font-bold rounded-lg text-white bg-primary hover:bg-blue-600 transition-all shadow-xl shadow-primary/30">
            {CTA.primaryCta}
          </button>
          <button className="inline-flex items-center justify-center px-8 py-4 border border-white/10 bg-transparent hover:bg-white/5 text-lg font-medium rounded-lg text-white transition-all">
            {CTA.secondaryCta}
          </button>
        </div>
      </div>
    </section>
  );
};

export default CallToAction;
