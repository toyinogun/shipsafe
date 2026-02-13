import React from "react";
import { INTEGRATIONS } from "../data/mockData";

interface IntegrationsProps {
  readonly className?: string;
}

export const Integrations: React.FC<IntegrationsProps> = ({
  className = "",
}) => {
  return (
    <section
      className={`py-20 relative overflow-hidden ${className}`}
      id="integrations"
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <div className="inline-block px-3 py-1 rounded-full bg-primary/10 text-primary text-xs font-bold mb-4 uppercase tracking-wider">
          Ecosystem
        </div>
        <h2 className="text-3xl font-bold text-white mb-12">
          Works where you work
        </h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
          {INTEGRATIONS.map((item) => (
            <div
              key={item.name}
              className="group p-6 bg-surface-dark border border-white/5 rounded-xl hover:border-white/20 transition-all cursor-pointer flex flex-col items-center"
            >
              <span className="material-symbols-outlined text-4xl text-slate-500 group-hover:text-white transition-colors mb-3">
                {item.icon}
              </span>
              <span className="text-slate-400 group-hover:text-white text-sm font-medium">
                {item.name}
              </span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default Integrations;
