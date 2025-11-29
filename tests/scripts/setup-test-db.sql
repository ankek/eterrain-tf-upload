-- Test Database Setup Script
-- Feature: 002-automated-testing
-- Purpose: Create isolated test database and user for integration tests

-- Create test database if it doesn't exist
CREATE DATABASE IF NOT EXISTS eterrain_test
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

-- Create test user if it doesn't exist
CREATE USER IF NOT EXISTS 'test_user'@'localhost' IDENTIFIED BY 'test_password';

-- Grant all privileges on test database to test user
GRANT ALL PRIVILEGES ON eterrain_test.* TO 'test_user'@'localhost';

-- Apply privilege changes
FLUSH PRIVILEGES;

-- Verify setup
SELECT 'Test database and user created successfully' AS Status;
