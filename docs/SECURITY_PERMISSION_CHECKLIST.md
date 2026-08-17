# Permission Security Checklist (Phase F5)

- [ ] The three account types — admin / operator / readonly — can log in and their profile includes `permissions`
- [ ] Operators only see orders/customer service/inventory/failed tasks for authorized stores
- [ ] Readonly write APIs return 40304
- [ ] Deep links without store permission return 404
- [ ] Settings PUT/test endpoints are admin-only
- [ ] User management APIs are admin-only; a user cannot disable themselves
- [ ] Operation logs do not contain secrets in plaintext
- [ ] The frontend PermissionGuard no-permission page is rendered in Chinese
- [ ] Demo accounts are documented in `docs/demo-dataset.permissions.json`
