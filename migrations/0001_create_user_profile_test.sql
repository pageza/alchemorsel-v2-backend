-- Test user profile table structure and constraints
DO $$
DECLARE
    test_user_id UUID := gen_random_uuid();
    test_profile_id UUID;
    test_dietary_id UUID;
    test_allergen_id UUID;
    test_history_id UUID;
BEGIN
    -- Test user_profiles table
    BEGIN
        -- Test valid insert
        INSERT INTO user_profiles (user_id, username, email)
        VALUES (test_user_id, 'testuser', 'test@example.com')
        RETURNING id INTO test_profile_id;
        
        -- Test username length constraint
        BEGIN
            INSERT INTO user_profiles (user_id, username, email)
            VALUES (test_user_id, 'ab', 'test2@example.com');
            RAISE EXCEPTION 'Username length constraint failed';
        EXCEPTION
            WHEN check_violation THEN
                NULL; -- Expected error
        END;
        
        -- Test email format constraint
        BEGIN
            INSERT INTO user_profiles (user_id, username, email)
            VALUES (test_user_id, 'testuser2', 'invalid-email');
            RAISE EXCEPTION 'Email format constraint failed';
        EXCEPTION
            WHEN check_violation THEN
                NULL; -- Expected error
        END;
        
        -- Test unique constraints
        BEGIN
            INSERT INTO user_profiles (user_id, username, email)
            VALUES (test_user_id, 'testuser', 'test3@example.com');
            RAISE EXCEPTION 'Username unique constraint failed';
        EXCEPTION
            WHEN unique_violation THEN
                NULL; -- Expected error
        END;
    END;

    -- Test dietary_preferences table
    BEGIN
        -- Test valid insert
        INSERT INTO dietary_preferences (user_id, preference_type)
        VALUES (test_user_id, 'vegan')
        RETURNING id INTO test_dietary_id;
        
        -- Test custom preference constraint
        BEGIN
            INSERT INTO dietary_preferences (user_id, preference_type)
            VALUES (test_user_id, 'custom');
            RAISE EXCEPTION 'Custom preference constraint failed';
        EXCEPTION
            WHEN check_violation THEN
                NULL; -- Expected error
        END;
        
        -- Test valid custom preference
        INSERT INTO dietary_preferences (user_id, preference_type, custom_name)
        VALUES (test_user_id, 'custom', 'Low FODMAP');
    END;

    -- Test allergens table
    BEGIN
        -- Test valid insert
        INSERT INTO allergens (user_id, allergen_name, severity_level)
        VALUES (test_user_id, 'Peanuts', 3)
        RETURNING id INTO test_allergen_id;
        
        -- Test severity level constraint
        BEGIN
            INSERT INTO allergens (user_id, allergen_name, severity_level)
            VALUES (test_user_id, 'Shellfish', 6);
            RAISE EXCEPTION 'Severity level constraint failed';
        EXCEPTION
            WHEN check_violation THEN
                NULL; -- Expected error
        END;
    END;

    -- Test profile_history table
    BEGIN
        -- Test valid insert
        INSERT INTO profile_history (user_id, field_name, old_value, new_value, changed_by)
        VALUES (test_user_id, 'username', 'oldname', 'newname', test_user_id)
        RETURNING id INTO test_history_id;
    END;

    -- Test updated_at triggers
    BEGIN
        -- Test user_profiles trigger
        UPDATE user_profiles SET username = 'updateduser' WHERE id = test_profile_id;
        ASSERT (SELECT updated_at > created_at FROM user_profiles WHERE id = test_profile_id),
            'updated_at trigger failed for user_profiles';
        
        -- Test dietary_preferences trigger
        UPDATE dietary_preferences SET custom_name = 'Updated Diet' WHERE id = test_dietary_id;
        ASSERT (SELECT updated_at > created_at FROM dietary_preferences WHERE id = test_dietary_id),
            'updated_at trigger failed for dietary_preferences';
        
        -- Test allergens trigger
        UPDATE allergens SET severity_level = 4 WHERE id = test_allergen_id;
        ASSERT (SELECT updated_at > created_at FROM allergens WHERE id = test_allergen_id),
            'updated_at trigger failed for allergens';
    END;

    -- Clean up test data
    DELETE FROM profile_history WHERE id = test_history_id;
    DELETE FROM allergens WHERE id = test_allergen_id;
    DELETE FROM dietary_preferences WHERE id = test_dietary_id;
    DELETE FROM user_profiles WHERE id = test_profile_id;

    RAISE NOTICE 'All migration tests passed successfully!';
END $$; 