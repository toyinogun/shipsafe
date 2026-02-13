import React from "react";
import { LOGO_CLOUD } from "../data/mockData";

interface LogoCloudProps {
  readonly className?: string;
}

export const LogoCloud: React.FC<LogoCloudProps> = ({ className = "" }) => {
  return (
    <section
      className={`border-y border-white/5 bg-surface-darker/50 py-10 ${className}`}
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <p className="text-center text-sm font-medium text-slate-500 mb-8 uppercase tracking-widest">
          {LOGO_CLOUD.heading}
        </p>
        <div className="flex flex-wrap justify-center items-center gap-8 md:gap-16 opacity-60 grayscale hover:grayscale-0 transition-all duration-500">
          {LOGO_CLOUD.logos.map((logo) => (
            <div
              key={logo.name}
              className="text-xl font-bold text-slate-300 flex items-center gap-2"
            >
              <span className="material-symbols-outlined">{logo.icon}</span>
              {logo.name}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default LogoCloud;
