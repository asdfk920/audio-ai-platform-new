-- Test data for audio separation tasks

-- Insert test tasks for user 123
INSERT INTO audio_separation_tasks (task_id, user_id, content_id, audio_url, status, progress, message, created_at, updated_at)
VALUES 
('task_123_1_001', 123, 1, 'http://localhost:8000/audio/test1.mp3', 'completed', 100, 'Completed', NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour'),
('task_123_2_002', 123, 2, 'http://localhost:8000/audio/test2.mp3', 'processing', 50, 'Processing...', NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes'),
('task_123_3_003', 123, 3, 'http://localhost:8000/audio/test3.mp3', 'pending', 0, 'Task created', NOW() - INTERVAL '10 minutes', NOW() - INTERVAL '10 minutes');

-- Insert tasks for other users (for testing user isolation)
INSERT INTO audio_separation_tasks (task_id, user_id, content_id, audio_url, status, progress, message, created_at, updated_at)
VALUES 
('task_456_1_004', 456, 1, 'http://localhost:8000/audio/test1.mp3', 'completed', 100, 'Completed', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours');
