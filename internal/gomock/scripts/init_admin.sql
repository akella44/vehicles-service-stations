DO $$
DECLARE
    hashed_password VARCHAR(255);
    new_employee_id INT;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM service_centers) THEN
        RAISE EXCEPTION 'Таблица service_centers пуста. Необходимо добавить хотя бы одну запись перед инициализацией администратора.';
    END IF;

    SELECT crypt('StrongPassword123!', gen_salt('bf')) INTO hashed_password;

    INSERT INTO employees (
        full_name, experience, age, salary, username, password_hash
    ) VALUES (
        'Admin User', 10, 40, 1000000.00, 'admin_user', hashed_password
    ) RETURNING employee_id INTO new_employee_id;

    INSERT INTO employee_service_center (
        employee_id, service_center_id, employee_role
    ) VALUES (
        new_employee_id,
        (SELECT service_center_id FROM service_centers ORDER BY RANDOM() LIMIT 1),
        'Administrator'
    );

    EXECUTE format('CREATE USER %I WITH PASSWORD %L', 'admin_user', 'StrongPassword123!');

    EXECUTE format('GRANT administrator TO %I', 'admin_user');

    EXECUTE format('ALTER ROLE %I WITH CREATEROLE', 'admin_user');

EXCEPTION
    WHEN others THEN
        RAISE NOTICE 'Произошла ошибка при инициализации администратора: %', SQLERRM;
        RAISE;
END;
$$ LANGUAGE plpgsql;
