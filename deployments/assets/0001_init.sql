DO $$ BEGIN
    IF to_regtype('loyalty_status') IS NULL THEN
        CREATE TYPE loyalty_status AS ENUM ('Bronze', 'Silver', 'Gold', 'Platinum');
    END IF;
END $$;

DO $$ BEGIN
    IF to_regtype('employee_role') IS NULL THEN
        CREATE TYPE employee_role AS ENUM ('Administrator', 'Analyst', 'Master', 'Manager');
    END IF;
END $$;

DO $$ BEGIN
    IF to_regtype('vehicle_type') IS NULL THEN
        CREATE TYPE vehicle_type AS ENUM ('Car', 'Moto');
    END IF;
END $$;

DO $$ BEGIN
    IF to_regtype('order_status') IS NULL THEN
        CREATE TYPE order_status AS ENUM ('Pending', 'In Progress','Completed', 'Cancelled');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS services (
    service_id SERIAL PRIMARY KEY,
    full_name VARCHAR(200) NOT NULL,
    description TEXT,
    vehicle_type vehicle_type NOT NULL,
    price NUMERIC(12, 2) NOT NULL CHECK (price >= 0)
);

CREATE TABLE IF NOT EXISTS customers (
    customer_id SERIAL PRIMARY KEY,
    phone_number VARCHAR(16) NOT NULL CHECK (phone_number ~ '^\+?\d{10,15}$'),
    full_name VARCHAR(100) NOT NULL,
    spent_money NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (spent_money >= 0),
    loyalty_status loyalty_status NOT NULL DEFAULT 'Bronze',
    bonus_points NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (bonus_points >= 0),
    last_bonus_charge_date TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS service_centers (
    service_center_id SERIAL PRIMARY KEY,
    full_address TEXT NOT NULL,
    city VARCHAR(50) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    phone_number VARCHAR(16) NOT NULL CHECK (phone_number ~ '^\+?\d{10,15}$'),
    employees_count INT NOT NULL DEFAULT 0 CHECK (employees_count >= 0)
);

CREATE TABLE IF NOT EXISTS employees (
    employee_id SERIAL PRIMARY KEY,
    full_name VARCHAR(100) NOT NULL,
    experience INT NOT NULL CHECK (experience >= 0),
    age INT NOT NULL CHECK (age > 0),
    salary NUMERIC(12, 2) NOT NULL CHECK (salary >= 0),
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(60) NOT NULL
);

CREATE TABLE IF NOT EXISTS employee_service_center (
    employee_id INT NOT NULL,
    service_center_id INT NOT NULL,
    employee_role employee_role NOT NULL,
    PRIMARY KEY (employee_id, service_center_id),
    FOREIGN KEY (employee_id) REFERENCES employees (employee_id) ON DELETE CASCADE,
    FOREIGN KEY (service_center_id) REFERENCES service_centers (service_center_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS orders (
    order_id SERIAL PRIMARY KEY,
    customer_id INT NOT NULL,
    service_center_id INT NOT NULL,
    manager_id INT NOT NULL,
    assigned_master_id INT NOT NULL,
    reassigned_master_id INT,
    creation_date DATE NOT NULL DEFAULT CURRENT_DATE,
    scheduled_date DATE NOT NULL CHECK (scheduled_date >= creation_date),
    status order_status DEFAULT 'Pending',
    total_cost NUMERIC(12, 2) DEFAULT 0 CHECK (total_cost >= 0),
    FOREIGN KEY (customer_id) REFERENCES customers (customer_id) ON DELETE CASCADE,
    FOREIGN KEY (service_center_id) REFERENCES service_centers (service_center_id) ON DELETE CASCADE,
    FOREIGN KEY (manager_id) REFERENCES employees (employee_id) ON DELETE SET NULL,
    FOREIGN KEY (assigned_master_id) REFERENCES employees (employee_id) ON DELETE SET NULL,
    FOREIGN KEY (reassigned_master_id) REFERENCES employees (employee_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS service_order (
    service_id INT NOT NULL,
    order_id INT NOT NULL,
    PRIMARY KEY (service_id, order_id),
    FOREIGN KEY (service_id) REFERENCES services (service_id) ON DELETE CASCADE,
    FOREIGN KEY (order_id) REFERENCES orders (order_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS stockpile (
    stockpile_id SERIAL PRIMARY KEY,
    full_address TEXT NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    phone_number VARCHAR(16) NOT NULL CHECK (phone_number ~ '^\+?\d{10,15}$')
);

CREATE TABLE IF NOT EXISTS spare_parts (
    part_id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    article_number INT NOT NULL,
    description TEXT,
    price NUMERIC(12, 2) NOT NULL CHECK (price >= 0),
    stock_quantity INT NOT NULL CHECK (stock_quantity >= 0),
    stockpile_id INT,
    FOREIGN KEY (stockpile_id) REFERENCES stockpile (stockpile_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS spare_part_order (
    part_id INT NOT NULL,
    order_id INT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    purchase_price NUMERIC(12, 2) NOT NULL CHECK (purchase_price >= 0),
    PRIMARY KEY (part_id, order_id),
    FOREIGN KEY (part_id) REFERENCES spare_parts (part_id) ON DELETE CASCADE,
    FOREIGN KEY (order_id) REFERENCES orders (order_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS receipts (
    receipt_id SERIAL PRIMARY KEY,
    order_id INT NOT NULL UNIQUE,
    bonus_points_spent NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (bonus_points_spent >= 0),
    total_paid NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (total_paid >= 0),
    receipt_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_receipts_orders
        FOREIGN KEY(order_id)
            REFERENCES orders(order_id)
            ON DELETE CASCADE
            ON UPDATE CASCADE
);

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_cron;

CREATE OR REPLACE VIEW bookings_by_date AS
SELECT
    scheduled_date,
    COUNT(*) AS total_bookings
FROM
    orders
GROUP BY
    scheduled_date
ORDER BY
    scheduled_date;

CREATE OR REPLACE VIEW most_demanded_services AS
SELECT
    s.service_id,
    s.full_name AS service_name,
    COUNT(so.order_id) AS demand_count
FROM
    services s
JOIN
    service_order so ON s.service_id = so.service_id
GROUP BY
    s.service_id, s.full_name
ORDER BY
    demand_count DESC;

CREATE OR REPLACE VIEW revenue_by_date AS
SELECT
    creation_date::date AS revenue_date,
    SUM(total_cost) AS total_revenue
FROM
    orders
WHERE
    status = 'Completed'
GROUP BY
    revenue_date
ORDER BY
    revenue_date;

CREATE OR REPLACE VIEW employee_performance AS
SELECT
    e.employee_id,
    e.full_name,
    COUNT(o.order_id) AS orders_handled,
    SUM(o.total_cost) AS total_revenue_generated
FROM
    employees e
JOIN
    orders o ON e.employee_id = o.assigned_master_id
WHERE
    o.status = 'Completed'
GROUP BY
    e.employee_id, e.full_name
ORDER BY
    orders_handled DESC;

CREATE OR REPLACE VIEW service_center_performance AS
SELECT
    sc.service_center_id,
    sc.full_address,
    COUNT(o.order_id) AS total_orders,
    SUM(o.total_cost) AS total_revenue
FROM
    service_centers sc
JOIN
    orders o ON sc.service_center_id = o.service_center_id
WHERE
    o.status = 'Completed'
GROUP BY
    sc.service_center_id, sc.full_address
ORDER BY
    total_orders DESC;

DO $$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'administrator') THEN
        CREATE ROLE administrator CREATEROLE;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'analyst') THEN
        CREATE ROLE analyst;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'master') THEN
        CREATE ROLE master;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'manager') THEN
        CREATE ROLE manager;
    END IF;
END $$;

GRANT manager TO administrator WITH ADMIN OPTION;
GRANT master TO administrator WITH ADMIN OPTION;
GRANT analyst TO administrator WITH ADMIN OPTION;

GRANT USAGE ON SCHEMA public TO administrator;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO administrator;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO administrator;

GRANT USAGE ON SCHEMA public TO analyst;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO analyst;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO analyst;
GRANT pg_read_server_files TO analyst;
GRANT pg_write_server_files TO analyst;

GRANT USAGE ON SCHEMA public TO master;
GRANT SELECT ON service_centers TO master;
GRANT SELECT ON employees TO master;
GRANT SELECT ON employee_service_center TO master;
GRANT SELECT, UPDATE ON orders TO master;
GRANT SELECT ON customers TO master;
GRANT SELECT ON services TO master;
GRANT SELECT ON service_order TO master;
GRANT SELECT ON spare_parts TO master;
GRANT SELECT ON spare_part_order TO master;
GRANT SELECT ON stockpile TO master;

GRANT USAGE ON SCHEMA public TO manager;
GRANT SELECT ON service_centers TO manager;
GRANT SELECT ON employees TO manager;
GRANT SELECT ON employee_service_center TO manager;
GRANT SELECT, INSERT, UPDATE ON customers TO manager;
GRANT SELECT, INSERT, UPDATE ON orders TO manager;
GRANT SELECT ON services TO manager;
GRANT SELECT, UPDATE ON service_order TO manager;
GRANT SELECT ON spare_parts TO manager;
GRANT SELECT, UPDATE ON spare_part_order TO manager;
GRANT SELECT ON stockpile TO manager;
GRANT SELECT, INSERT, UPDATE ON receipts TO manager; 
GRANT USAGE ON SEQUENCE customers_customer_id_seq TO manager;
GRANT USAGE ON SEQUENCE orders_order_id_seq TO manager;
GRANT USAGE ON SEQUENCE receipts_receipt_id_seq TO manager;

ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE employees ENABLE ROW LEVEL SECURITY;
ALTER TABLE employee_service_center ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_centers ENABLE ROW LEVEL SECURITY;

CREATE POLICY admin_employees_policies ON employees
    FOR ALL
    TO administrator
    USING (true);

CREATE POLICY admin_employee_service_center_policies ON employee_service_center
    FOR ALL
    TO administrator
    USING (true);

CREATE POLICY admin_service_centers_select_policy ON service_centers
    FOR SELECT
    TO administrator
    USING (true);

CREATE POLICY admin_service_centers_update_policy ON service_centers
    FOR UPDATE
    TO administrator
    USING (
        service_center_id IN (
            SELECT esc.service_center_id
            FROM employee_service_center esc
            JOIN employees e ON esc.employee_id = e.employee_id
            WHERE e.username = current_user
              AND esc.employee_role = 'Administrator'
        )
    );

CREATE POLICY admin_customers_select_policy ON customers
    FOR SELECT
    TO administrator
    USING (true);

CREATE POLICY admin_customers_insert_policy ON customers
    FOR INSERT
    TO administrator
    WITH CHECK (true);

CREATE POLICY admin_customers_update_policy ON customers
    FOR UPDATE
    TO administrator
    USING (
        EXISTS (
            SELECT 1
            FROM orders o
            JOIN employee_service_center esc ON o.service_center_id = esc.service_center_id
            JOIN employees e ON esc.employee_id = e.employee_id
            WHERE o.customer_id = customers.customer_id
              AND e.username = current_user
        )
    );

CREATE POLICY master_employees_select_policy ON employees
    FOR SELECT
    TO master
    USING (
        username = current_user
    );

CREATE POLICY master_employee_service_center_select_policy ON employee_service_center
    FOR SELECT
    TO master
    USING (
        employee_id = (
            SELECT employee_id
            FROM employees
            WHERE username = current_user
        )
    );

CREATE POLICY master_orders_select_policy ON orders
    FOR SELECT
    TO master
    USING (
        assigned_master_id = (
            SELECT employee_id
            FROM employees
            WHERE username = current_user
        ) OR
        reassigned_master_id = (
            SELECT employee_id
            FROM employees
            WHERE username = current_user
        )::integer --here can be null
    );

CREATE POLICY master_orders_update_policy ON orders
    FOR UPDATE
    TO master
    USING (
        assigned_master_id = (
            SELECT employee_id
            FROM employees
            WHERE username = current_user
        ) OR
        reassigned_master_id = (
            SELECT employee_id
            FROM employees
            WHERE username = current_user
        )::integer --here can be null
    );

CREATE POLICY master_customers_select_policy ON customers
    FOR SELECT
    TO master
    USING (
        EXISTS (
            SELECT 1
            FROM orders o
            WHERE o.customer_id = customers.customer_id
              AND (o.assigned_master_id = (
                  SELECT employee_id
                  FROM employees
                  WHERE username = current_user
              ) OR 
              o.reassigned_master_id = (
                SELECT employee_id
                FROM employees
                WHERE username = current_user
              )::integer
            )
        )
    );

CREATE POLICY master_service_centers_select_policy ON service_centers
    FOR SELECT
    TO master
    USING (
        service_center_id IN (
            SELECT esc.service_center_id
            FROM employee_service_center esc
            JOIN employees e ON esc.employee_id = e.employee_id
            WHERE e.username = current_user
        )
    );

-- CREATE OR REPLACE VIEW service_center_staff AS
-- SELECT employee_id 
-- FROM employee_service_center
-- WHERE service_center_id IN (
--     SELECT esc_mgr.service_center_id
--     FROM employee_service_center esc_mgr
--     JOIN employees e_mgr ON esc_mgr.employee_id = e_mgr.employee_id
--     WHERE e_mgr.username = current_user
-- );

-- GRANT SELECT ON service_center_staff TO manager;

CREATE OR REPLACE FUNCTION get_service_center_staff()
RETURNS TABLE(employee_id INTEGER)
LANGUAGE SQL
SECURITY DEFINER
AS $$
    SELECT es.employee_id
    FROM employee_service_center es
    WHERE es.service_center_id IN (
        SELECT esc_mgr.service_center_id
        FROM employee_service_center esc_mgr
        JOIN employees e_mgr ON esc_mgr.employee_id = e_mgr.employee_id
        WHERE e_mgr.username = session_user
    );
$$;

ALTER FUNCTION get_service_center_staff() OWNER TO administrator;
GRANT EXECUTE ON FUNCTION get_service_center_staff() TO manager;

CREATE POLICY manager_employees_select_policy ON employees 
    FOR SELECT
    TO manager
    USING (
        employee_id IN (
            SELECT employee_id FROM get_service_center_staff()
        )
    );

CREATE POLICY manager_service_centers_select_policy ON service_centers
    FOR SELECT
    TO manager
    USING (
        service_center_id IN (
            SELECT esc.service_center_id
            FROM employee_service_center esc
            JOIN employees e ON esc.employee_id = e.employee_id
            WHERE e.username = current_user
        )
    );

CREATE POLICY manager_employee_service_center_select_policy ON employee_service_center
    FOR SELECT
    TO manager
    USING (
        employee_id IN (
            SELECT employee_id FROM get_service_center_staff()
        )
    );

CREATE POLICY manager_orders_select_policy ON orders
    FOR SELECT
    TO manager
    USING (
        service_center_id IN (
            SELECT esc.service_center_id
            FROM employee_service_center esc
            JOIN employees e ON esc.employee_id = e.employee_id
            WHERE e.username = current_user
        )
    );

CREATE POLICY manager_orders_insert_policy ON orders
    FOR INSERT
    TO manager
    WITH CHECK (
        manager_id = (
            SELECT employee_id
            FROM employees
            WHERE username = current_user
        )
    );

CREATE POLICY manager_orders_update_policy ON orders
    FOR UPDATE
    TO manager
    USING (
        manager_id = (
            SELECT employee_id
            FROM employees
            WHERE username = current_user
        )
    );

CREATE POLICY manager_customers_select_policy ON customers
    FOR SELECT
    TO manager
    USING (true);

CREATE POLICY manager_customers_update_policy ON customers
    FOR UPDATE
    TO manager
    USING (true);

CREATE POLICY manager_customers_insert_policy ON customers
    FOR INSERT
    TO manager
    WITH CHECK (true);

CREATE POLICY analyst_select_customers_policy ON customers
    FOR SELECT
    TO analyst
    USING (true);

CREATE POLICY analyst_select_service_centers_policy ON service_centers
    FOR SELECT
    TO analyst
    USING (true);

CREATE POLICY analyst_select_employees_policy ON employees
    FOR SELECT
    TO analyst
    USING (true);

CREATE POLICY analyst_select_employee_service_center_policy ON employee_service_center
    FOR SELECT
    TO analyst
    USING (true);

CREATE POLICY analyst_select_orders_policy ON orders
    FOR SELECT
    TO analyst
    USING (true);

CREATE OR REPLACE FUNCTION create_user(
		p_full_name VARCHAR(100),
		p_experience INT,
		p_age INT,
		p_salary NUMERIC(10, 2),
		p_username VARCHAR(50),
		p_password TEXT,
		p_role employee_role,
		p_service_center_id INT
	)
	RETURNS VOID AS $$
	DECLARE
		hashed_password VARCHAR(255);
		new_employee_id INT;
	BEGIN
		SELECT crypt(p_password, gen_salt('bf')) INTO hashed_password;

		INSERT INTO employees (
			full_name, experience, age, salary, username, password_hash
		) VALUES (
			p_full_name, p_experience, p_age, p_salary, p_username, hashed_password
		) RETURNING employee_id INTO new_employee_id;

		INSERT INTO employee_service_center (
			employee_id, service_center_id, employee_role
		) VALUES (
			new_employee_id, p_service_center_id, p_role
		);

		EXECUTE format('CREATE USER %I WITH PASSWORD %L', p_username, p_password);

		EXECUTE format('GRANT %I TO %I', LOWER(p_role::TEXT), p_username);
	END;
	$$ LANGUAGE plpgsql;

REVOKE EXECUTE ON FUNCTION create_user(VARCHAR, INT, INT, NUMERIC, VARCHAR, TEXT, employee_role, INT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION create_user(VARCHAR, INT, INT, NUMERIC, VARCHAR, TEXT, employee_role, INT) TO administrator;

CREATE OR REPLACE FUNCTION update_loyalty_status() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.spent_money >= 100000 THEN
        NEW.loyalty_status := 'Platinum';
    ELSIF NEW.spent_money >= 50000 THEN
        NEW.loyalty_status := 'Gold';
    ELSIF NEW.spent_money >= 10000 THEN
        NEW.loyalty_status := 'Silver';
    ELSE
        NEW.loyalty_status := 'Bronze';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER loyalty_status_update_trigger
AFTER UPDATE OF spent_money
ON customers
FOR EACH ROW
EXECUTE FUNCTION update_loyalty_status();

CREATE OR REPLACE FUNCTION update_order_total_cost() RETURNS TRIGGER AS $$
BEGIN
    UPDATE orders
    SET total_cost = (
        COALESCE((
            SELECT SUM(s.price)
            FROM service_order so
            JOIN services s ON so.service_id = s.service_id
            WHERE so.order_id = NEW.order_id
        ), 0)
        +
        COALESCE((
            SELECT SUM(spo.purchase_price * spo.quantity)
            FROM spare_part_order spo
            WHERE spo.order_id = NEW.order_id
        ), 0)
    )
    WHERE order_id = NEW.order_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_total_cost_service_order_trigger
AFTER INSERT ON service_order
FOR EACH ROW
EXECUTE FUNCTION update_order_total_cost();

CREATE TRIGGER update_total_cost_spare_part_order_trigger
AFTER INSERT ON spare_part_order
FOR EACH ROW
EXECUTE FUNCTION update_order_total_cost();

CREATE OR REPLACE FUNCTION update_customer_on_receipt()
RETURNS TRIGGER AS $$
DECLARE
    v_customer_id INT;
BEGIN
    SELECT customer_id INTO v_customer_id
    FROM orders
    WHERE order_id = NEW.order_id;
    
    IF v_customer_id IS NULL THEN
        RAISE EXCEPTION 'Order with order_id % does not exist.', NEW.order_id;
    END IF;
    
    UPDATE customers
    SET
        spent_money = spent_money + NEW.total_paid,
        bonus_points = bonus_points - NEW.bonus_points_spent
    WHERE customer_id = v_customer_id;

    IF (SELECT bonus_points FROM customers WHERE customer_id = v_customer_id) < 0 THEN
        RAISE EXCEPTION 'Customer % has insufficient bonus points.', v_customer_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_customer_after_receipt_trigger
AFTER INSERT ON receipts
FOR EACH ROW
EXECUTE FUNCTION update_customer_on_receipt();

CREATE OR REPLACE FUNCTION update_bonus_points()
RETURNS TRIGGER AS $$
DECLARE
    difference NUMERIC(12, 2);
    points_change NUMERIC(12, 2);
BEGIN
    difference := NEW.spent_money - OLD.spent_money;
    
    IF difference > 0 THEN
        points_change := difference / 10;
        NEW.bonus_points := OLD.bonus_points + points_change;
    
    ELSIF difference < 0 THEN
        points_change := difference / 10;
        NEW.bonus_points := OLD.bonus_points + points_change;
        
        IF NEW.bonus_points < 0 THEN
            NEW.bonus_points := 0;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_bonus_points_trigger
BEFORE UPDATE OF spent_money ON customers
FOR EACH ROW
EXECUTE FUNCTION update_bonus_points();

CREATE OR REPLACE FUNCTION update_last_bonus_charge_date()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.bonus_points < OLD.bonus_points THEN
        NEW.last_bonus_charge_date := CURRENT_TIMESTAMP;

    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_last_bonus_charge_date_trigger
BEFORE UPDATE OF bonus_points ON customers
FOR EACH ROW
EXECUTE FUNCTION update_last_bonus_charge_date();

CREATE OR REPLACE FUNCTION update_employees_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE service_centers
    SET employees_count = employees_count + 1
    WHERE service_center_id = NEW.service_center_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_employees_count_trigger
AFTER INSERT
ON employee_service_center
FOR EACH ROW
EXECUTE FUNCTION update_employees_count();

CREATE OR REPLACE FUNCTION check_spare_part_stock()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM spare_parts
        WHERE part_id = NEW.part_id
    ) THEN
        RAISE EXCEPTION 'Detail with part_id % does not exist.', NEW.part_id;
    END IF;

    PERFORM 1
    FROM spare_parts
    WHERE part_id = NEW.part_id
      AND stock_quantity >= NEW.quantity;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Insufficient stock for part_id %. Available: %, Requested: %.',
                        NEW.part_id,
                        (SELECT stock_quantity FROM spare_parts WHERE part_id = NEW.part_id),
                        NEW.quantity;
    END IF;

    UPDATE spare_parts
    SET stock_quantity = stock_quantity - NEW.quantity
    WHERE part_id = NEW.part_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER check_spare_part_stock_trigger
BEFORE INSERT ON spare_part_order
FOR EACH ROW
EXECUTE FUNCTION check_spare_part_stock();

CREATE OR REPLACE FUNCTION reset_inactive_bonus_points()
RETURNS VOID AS $$
BEGIN
    UPDATE customers
    SET bonus_points = 0,
        last_bonus_charge_date = CURRENT_TIMESTAMP
    WHERE last_bonus_charge_date < (CURRENT_TIMESTAMP - INTERVAL '1 year')
      AND bonus_points > 0;
    
    RAISE NOTICE 'Bonus points reset for customers with no activity in the last year.';
END;
$$ LANGUAGE plpgsql;

SELECT cron.schedule(
    'reset_bonus_points',
    '0 0 * * *',
    'SELECT reset_inactive_bonus_points();'
);