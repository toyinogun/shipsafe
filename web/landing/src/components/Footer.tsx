import React from "react";
import { FOOTER } from "../data/mockData";

interface FooterProps {
  readonly className?: string;
}

export const Footer: React.FC<FooterProps> = ({ className = "" }) => {
  return (
    <footer
      className={`bg-surface-darker border-t border-white/10 pt-16 pb-8 ${className}`}
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mb-12">
          {/* Brand */}
          <div className="col-span-2 md:col-span-1">
            <div className="flex items-center gap-2 mb-4">
              <div className="w-6 h-6 rounded-sm bg-primary flex items-center justify-center text-white font-bold text-xs">
                S
              </div>
              <span className="font-bold text-lg text-white">ShipSafe</span>
            </div>
            <p className="text-slate-500 text-sm">{FOOTER.tagline}</p>
          </div>

          {/* Link columns */}
          {FOOTER.columns.map((col) => (
            <div key={col.title}>
              <h4 className="text-white font-semibold mb-4">{col.title}</h4>
              <ul className="space-y-2 text-sm text-slate-400">
                {col.links.map((link) => (
                  <li key={link.label}>
                    <a href={link.href} className="hover:text-primary">
                      {link.label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Bottom bar */}
        <div className="border-t border-white/5 pt-8 flex flex-col md:flex-row justify-between items-center gap-4">
          <p className="text-slate-600 text-xs">{FOOTER.copyright}</p>
          <div className="flex gap-4">
            <a
              href="#"
              className="text-slate-500 hover:text-white transition-colors"
              aria-label="Twitter"
            >
              <span className="material-symbols-outlined text-lg">share</span>
            </a>
            <a
              href="#"
              className="text-slate-500 hover:text-white transition-colors"
              aria-label="GitHub"
            >
              <span className="material-symbols-outlined text-lg">code</span>
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
