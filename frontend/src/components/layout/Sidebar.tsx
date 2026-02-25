import { NavLink } from 'react-router-dom';
import {
  LayoutDashboard,
  ShieldAlert,
  Search,
  Users,
  BookOpen,
  BookText,
  Radio,
  Settings,
  UserCog,
} from 'lucide-react';
import { useAuthStore } from '@/store/authStore';
import type { UserRole } from '@/types';

interface NavItem {
  to: string;
  label: string;
  icon: React.ReactNode;
  minRole: UserRole[];
}

const navItems: NavItem[] = [
  { to: '/', label: 'Dashboard', icon: <LayoutDashboard size={20} />, minRole: ['superadmin', 'admin', 'moderator'] },
  { to: '/detected-spam', label: 'Detected Spam', icon: <ShieldAlert size={20} />, minRole: ['superadmin', 'admin', 'moderator'] },
  { to: '/spam-check', label: 'Spam Check', icon: <Search size={20} />, minRole: ['superadmin', 'admin'] },
  { to: '/approved-users', label: 'Approved Users', icon: <Users size={20} />, minRole: ['superadmin', 'admin', 'moderator'] },
  { to: '/samples', label: 'Samples', icon: <BookOpen size={20} />, minRole: ['superadmin', 'admin'] },
  { to: '/dictionary', label: 'Dictionary', icon: <BookText size={20} />, minRole: ['superadmin', 'admin'] },
  { to: '/channels', label: 'Channels', icon: <Radio size={20} />, minRole: ['superadmin'] },
  { to: '/settings', label: 'Settings', icon: <Settings size={20} />, minRole: ['superadmin', 'admin'] },
  { to: '/admin-users', label: 'Admin Users', icon: <UserCog size={20} />, minRole: ['superadmin'] },
];

export default function Sidebar() {
  const user = useAuthStore((s) => s.user);
  const role = user?.role;

  const filteredItems = navItems.filter((item) => role && item.minRole.includes(role as UserRole));

  return (
    <aside className="w-64 bg-white border-r border-gray-200 min-h-screen">
      <div className="p-6">
        <h1 className="text-xl font-bold text-primary-600">TG-Spam Admin</h1>
        <p className="text-xs text-gray-500 mt-1">Anti-Spam Management</p>
      </div>
      <nav className="px-3">
        {filteredItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-primary-50 text-primary-700'
                  : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
              }`
            }
          >
            {item.icon}
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}
