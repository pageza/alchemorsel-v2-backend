DO $$
DECLARE
    test_user_id UUID;
    test_profile_id UUID;
    test_dietary_id UUID;
    test_allergen_id UUID;
    test_appliance_id UUID;
BEGIN
    INSERT INTO users (name, email, password_hash)
    VALUES ('Test User Enhanced', 'test_enhanced@example.com', 'hashed_password')
    RETURNING id INTO test_user_id;
    
    INSERT INTO user_profiles (user_id, username, email, bio, cooking_ability_level)
    VALUES (test_user_id, 'testuser_enhanced', 'test_enhanced@example.com', 'Test bio', 'intermediate')
    RETURNING id INTO test_profile_id;
    
    INSERT INTO dietary_preferences (user_id, preference_type)
    VALUES (test_user_id, 'vegan')
    RETURNING id INTO test_dietary_id;
    
    INSERT INTO allergens (user_id, allergen_name, severity_level)
    VALUES (test_user_id, 'Peanuts', 4)
    RETURNING id INTO test_allergen_id;
    
    INSERT INTO user_appliances (user_id, appliance_type)
    VALUES (test_user_id, 'oven')
    RETURNING id INTO test_appliance_id;
    
    INSERT INTO user_appliances (user_id, appliance_type, custom_name)
    VALUES (test_user_id, 'custom', 'Pizza Stone');
    
    ASSERT (SELECT cooking_ability_level FROM user_profiles WHERE id = test_profile_id) = 'intermediate',
        'Cooking ability level not set correctly';
    
    ASSERT (SELECT COUNT(*) FROM user_appliances WHERE user_id = test_user_id) = 2,
        'Appliances not inserted correctly';
    
    DELETE FROM user_appliances WHERE user_id = test_user_id;
    DELETE FROM allergens WHERE id = test_allergen_id;
    DELETE FROM dietary_preferences WHERE id = test_dietary_id;
    DELETE FROM user_profiles WHERE id = test_profile_id;
    DELETE FROM users WHERE id = test_user_id;
    
    RAISE NOTICE 'Enhanced user profile test completed successfully';
END;
$$;
