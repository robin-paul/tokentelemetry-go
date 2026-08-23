import React, { useEffect, useState } from 'react';
import {
  LayoutDashboard,
  Terminal,
  FolderGit2,
  LineChart,
  Settings as SettingsIcon,
  Sun,
  Moon,
  Activity,
  Radio,
} from 'lucide-react';
import { apiFetch, subscribeEvents } from '../lib/api';

interface NavigationProps {
  currentPath?: string;
}

export const Navigation: React.FC<NavigationProps> = ({ currentPath: initialPath }) => {
  const [currentPath, setCurrentPath] = useState(initialPath || '/');
  const [agents, setAgents] = useState<string[]>([]);
  const [theme, setTheme] = useState<'dark' | 'light'>('dark');
  const [liveConnected, setLiveConnected] = useState(true);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      setCurrentPath(window.location.pathname);
      const savedTheme = (localStorage.getItem('tt_theme') as 'dark' | 'light') || 'dark';
      setTheme(savedTheme);
      document.documentElement.setAttribute('data-theme', savedTheme);
    }

    apiFetch<string[]>('/agents')
      .then((data) => setAgents(data || []))
      .catch(() => {});

    const unsubscribe = subscribeEvents(() => {
      setLiveConnected(true);
    });

    return () => unsubscribe();
  }, []);

  const toggleTheme = () => {
    const nextTheme = theme === 'dark' ? 'light' : 'dark';
    setTheme(nextTheme);
    localStorage.setItem('tt_theme', nextTheme);
    document.documentElement.setAttribute('data-theme', nextTheme);
  };

  const navItems = [
    { label: 'Overview', href: '/', icon: LayoutDashboard },
    { label: 'Sessions', href: '/sessions', icon: Terminal },
    { label: 'Projects', href: '/projects', icon: FolderGit2 },
    { label: 'Analytics', href: '/analytics', icon: LineChart },
    { label: 'Settings', href: '/settings', icon: SettingsIcon },
  ];

  return (
    <aside className="w-64 border-r border-white/10 bg-[#0d1017] flex flex-col justify-between h-screen sticky top-0 select-none">
      <div>
        {/* Brand Header */}
        <div className="p-5 flex items-center gap-3 border-b border-white/10">
          <div className="w-8 h-8 rounded-lg bg-blue-500/20 border border-blue-500/40 flex items-center justify-center text-blue-400 font-bold">
            <Activity className="w-5 h-5" />
          </div>
          <div>
            <div className="font-semibold text-sm tracking-wide text-white">TokenTelemetry</div>
            <div className="text-[11px] text-gray-400 flex items-center gap-1.5">
              <span className={`w-1.5 h-1.5 rounded-full ${liveConnected ? 'bg-emerald-400 animate-pulse' : 'bg-amber-400'}`} />
              {liveConnected ? 'Live Telemetry' : 'Connecting...'}
            </div>
          </div>
        </div>

        {/* Primary Navigation */}
        <nav className="p-3 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive =
              item.href === '/'
                ? currentPath === '/'
                : currentPath.startsWith(item.href);

            return (
              <a
                key={item.href}
                href={item.href}
                className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-xs font-medium transition-all ${
                  isActive
                    ? 'bg-blue-600/20 text-blue-400 border border-blue-500/30'
                    : 'text-gray-400 hover:text-white hover:bg-white/5 border border-transparent'
                }`}
              >
                <Icon className={`w-4 h-4 ${isActive ? 'text-blue-400' : 'text-gray-400'}`} />
                <span>{item.label}</span>
              </a>
            );
          })}
        </nav>

        {/* Detected Agent Count Pill */}
        <div className="mx-3 mt-4 p-3 rounded-lg bg-white/[0.03] border border-white/5">
          <div className="text-[11px] text-gray-400 uppercase font-semibold tracking-wider flex items-center justify-between mb-2">
            <span>Ecosystem</span>
            <Radio className="w-3.5 h-3.5 text-blue-400" />
          </div>
          <div className="text-xs text-gray-300">
            <span className="font-bold text-white">{agents.length}</span> Active Agents Detected
          </div>
        </div>
      </div>

      {/* Footer Controls */}
      <div className="p-4 border-t border-white/10 flex items-center justify-between">
        <div className="text-[11px] text-gray-500">v1.0.0 (Go Single-Binary)</div>
        <button
          onClick={toggleTheme}
          title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
          className="p-1.5 rounded-md text-gray-400 hover:text-white hover:bg-white/10 transition-colors"
        >
          {theme === 'dark' ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
        </button>
      </div>
    </aside>
  );
};
