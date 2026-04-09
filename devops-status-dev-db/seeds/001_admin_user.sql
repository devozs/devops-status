-- Default admin user (password: admin123)
-- bcrypt hash for "admin123"
INSERT INTO admin_users (username, password_hash, role, is_active)
VALUES ('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin', true)
ON CONFLICT (username) DO NOTHING;
