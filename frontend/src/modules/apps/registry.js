export const appRegistry = [
	{ id: 'dashboard', title: 'Dashboard', route: '/dashboard', icon: '⌁', permissions: ['system:read'], window: { width: 980, height: 680, minWidth: 720, minHeight: 420 } },
	{ id: 'files', title: 'Arquivos', route: '/files', icon: '▣', permissions: ['files:read'], window: { width: 1080, height: 720, minWidth: 760, minHeight: 460 } },
	{ id: 'system', title: 'Sistema', route: '/system', icon: '◉', permissions: ['system:read'], window: { width: 960, height: 680, minWidth: 720, minHeight: 420 } },
	{ id: 'samba', title: 'SMB', route: '/samba', icon: '⇄', permissions: ['shares:read'], window: { width: 920, height: 650, minWidth: 700, minHeight: 420 } },
	{ id: 'users', title: 'Usuários', route: '/users', icon: '◌', permissions: ['users:admin'], window: { width: 920, height: 650, minWidth: 700, minHeight: 420 } },
	{ id: 'settings', title: 'Ajustes', route: '/settings', icon: '⚙', permissions: ['settings:read'], window: { width: 760, height: 560, minWidth: 620, minHeight: 420 } }
];

export function findAppByRoute(route) {
	return appRegistry.find((app) => route === app.route || route.startsWith(`${app.route}/`)) || appRegistry[0];
}

export function findAppById(id) {
	return appRegistry.find((app) => app.id === id);
}
