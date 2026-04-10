-- Default admin user (password: admin123)
-- bcrypt hash for "admin123"
INSERT INTO admin_users (username, password_hash, role, is_active)
VALUES ('admin', '$2a$10$pH2LLV04TvRr1uF/dF8AYOTu9xS22mG2E8vGu7izPGMlwbDf9k.nO', 'admin', true)
ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash;
