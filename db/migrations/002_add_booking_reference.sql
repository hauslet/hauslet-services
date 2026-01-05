-- -- +goose Up
-- -- Add booking_reference column to bookings table
-- ALTER TABLE bookings
-- ADD COLUMN booking_reference VARCHAR(20);

-- -- Generate references for existing bookings
-- -- This uses a simple sequential approach for existing data
-- -- New bookings will use the crypto-random generation
-- -- +goose StatementBegin
-- DO $$
-- DECLARE
--     booking_record RECORD;
--     new_reference VARCHAR(20);
--     reference_exists BOOLEAN;
--     retry_count INT;
-- BEGIN
--     FOR booking_record IN SELECT id FROM bookings WHERE booking_reference IS NULL ORDER BY created_at
--     LOOP
--         retry_count := 0;
--         reference_exists := TRUE;
        
--         WHILE reference_exists AND retry_count < 10 LOOP
--             -- Generate reference: H + 6 random digits + '-' + 3 random letters
--             new_reference := 'H' || 
--                            LPAD(FLOOR(RANDOM() * 900000 + 100000)::TEXT, 6, '0') || 
--                            '-' ||
--                            CHR(65 + FLOOR(RANDOM() * 26)::INT) ||
--                            CHR(65 + FLOOR(RANDOM() * 26)::INT) ||
--                            CHR(65 + FLOOR(RANDOM() * 26)::INT);
            
--             -- Check if reference exists
--             SELECT EXISTS(SELECT 1 FROM bookings WHERE booking_reference = new_reference) INTO reference_exists;
            
--             retry_count := retry_count + 1;
--         END LOOP;
        
--         IF NOT reference_exists THEN
--             UPDATE bookings SET booking_reference = new_reference WHERE id = booking_record.id;
--         ELSE
--             RAISE EXCEPTION 'Failed to generate unique reference for booking %', booking_record.id;
--         END IF;
--     END LOOP;
-- END $$;
-- -- +goose StatementEnd

-- -- Now make the column NOT NULL
-- ALTER TABLE bookings
-- ALTER COLUMN booking_reference SET NOT NULL;

-- -- Add unique constraint
-- CREATE UNIQUE INDEX idx_bookings_booking_reference ON bookings(booking_reference);

-- -- Add index for faster lookups by reference
-- CREATE INDEX idx_bookings_booking_reference_lookup ON bookings(booking_reference) WHERE deleted_at IS NULL;

-- -- +goose Down
-- DROP INDEX IF EXISTS idx_bookings_booking_reference_lookup;
-- DROP INDEX IF EXISTS idx_bookings_booking_reference;
-- ALTER TABLE bookings
-- DROP COLUMN IF EXISTS booking_reference;
