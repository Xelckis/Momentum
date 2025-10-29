-- Migration to add the column
ALTER TABLE job_types 
ADD COLUMN custom_field_definitions JSONB;
