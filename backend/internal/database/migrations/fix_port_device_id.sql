-- Fix Port device_id column to support NULL and increase length
-- This migration fixes the foreign key constraint issues

USE openstack_monitor;

-- Drop foreign key constraint if exists
ALTER TABLE ports DROP FOREIGN KEY IF EXISTS fk_ports_instance;

-- Modify device_id column to allow NULL and increase length
ALTER TABLE ports MODIFY COLUMN device_id VARCHAR(255) NULL;

-- Note: We don't recreate the foreign key constraint because device_id
-- can reference either instances or routers, and routers don't have
-- a foreign key relationship to instances.

