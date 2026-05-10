// src/lib/api/samba.js
// Wrapper das rotas /api/v1/samba/*
import { api } from './client.js';

export const sambaApi = {
	// Estado completo (shares + users + status).
	getState: () => api('/samba/state'),

	// Shares
	listShares:   ()        => api('/samba/shares'),
	getShare:     (name)    => api(`/samba/shares/${encodeURIComponent(name)}`),
	createShare:  (share)   => api('/samba/shares', { method: 'POST', body: share }),
	updateShare:  (name, share) => api(`/samba/shares/${encodeURIComponent(name)}`, { method: 'PUT', body: share }),
	deleteShare:  (name)    => api(`/samba/shares/${encodeURIComponent(name)}`, { method: 'DELETE' }),

	// Users
	listUsers:    ()        => api('/samba/users'),
	createUser:   (username, password) =>
		api('/samba/users', { method: 'POST', body: { username, password } }),
	setUserPassword: (name, password) =>
		api(`/samba/users/${encodeURIComponent(name)}/password`, { method: 'POST', body: { password } }),
	setUserDisabled: (name, disabled) =>
		api(`/samba/users/${encodeURIComponent(name)}`, { method: 'PATCH', body: { disabled } }),
	deleteUser:   (name)    => api(`/samba/users/${encodeURIComponent(name)}`, { method: 'DELETE' })
};
