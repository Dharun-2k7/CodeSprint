-- Make only one user admin. Replace your@email.com with your email, then run:
--   docker exec -i codesprint_db psql -U codesprint -d codesprint < scripts/set-admin.sql

UPDATE users SET is_admin = false;
UPDATE users SET is_admin = true WHERE email = 'your@email.com';
