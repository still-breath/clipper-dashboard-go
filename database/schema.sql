-- Create database
CREATE DATABASE cctv_system;

-- Use the database
\c cctv_system;

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Courts table
CREATE TABLE courts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Booking hours table
CREATE TABLE booking_hours (
    id SERIAL PRIMARY KEY,
    court_id INTEGER NOT NULL REFERENCES courts(id) ON DELETE CASCADE,
    date_start TIMESTAMP WITH TIME ZONE NOT NULL,
    date_end TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Clips table
CREATE TABLE clips (
    id SERIAL PRIMARY KEY,
    booking_hour_id INTEGER NOT NULL REFERENCES booking_hours(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT,
    mime_type VARCHAR(100),
    duration_seconds INTEGER,
    camera_name VARCHAR(255),
    upload_status VARCHAR(50) DEFAULT 'uploaded',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_courts_name ON courts(name);
CREATE INDEX idx_booking_hours_court_id ON booking_hours(court_id);
CREATE INDEX idx_booking_hours_date_start ON booking_hours(date_start);
CREATE INDEX idx_booking_hours_date_end ON booking_hours(date_end);
CREATE INDEX idx_clips_booking_hour_id ON clips(booking_hour_id);
CREATE INDEX idx_clips_created_at ON clips(created_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_courts_updated_at BEFORE UPDATE ON courts 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    
CREATE TRIGGER update_booking_hours_updated_at BEFORE UPDATE ON booking_hours 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    
CREATE TRIGGER update_clips_updated_at BEFORE UPDATE ON clips 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample data
INSERT INTO courts (name, description) VALUES 
('Lapangan 1 Kiri', 'Left side of Court 1'),
('Lapangan 1 Kanan', 'Right side of Court 1'),
('Lapangan 2 Kiri', 'Left side of Court 2'),
('Lapangan 2 Kanan', 'Right side of Court 2');