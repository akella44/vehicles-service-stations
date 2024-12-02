package gomock

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"regexp"

	"github.com/Masterminds/squirrel"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/jackc/pgx/v5/pgxpool"

	"vehicles-service-stations/internal/model"
	"vehicles-service-stations/internal/utils"
)

type Pair[T any] struct {
	First  T
	Second T
}

func CreateServiceCenters(ctx context.Context, db *pgxpool.Pool, cityCount int, serviceCenterCount int) error {

	set := make(map[string]struct{})

	for i := 0; i < cityCount; i++ {
		set[gofakeit.City()] = struct{}{}
	}

	cities := make([]string, 0, len(set))
	for key := range set {
		cities = append(cities, key)
	}

	for i := 0; i < serviceCenterCount; i++ {
		city := cities[rand.IntN(len(cities))]
		re := regexp.MustCompile(`, (\w+),`)
		updatedAddress := re.ReplaceAllString(gofakeit.Address().Address, fmt.Sprintf(", %s,", city))
		postalCode := gofakeit.Zip()
		phone := "+" + gofakeit.Phone()
		_, err := db.Exec(ctx, `INSERT INTO service_centers (full_address, city, postal_code, phone_number)
			VALUES ($1, $2, $3, $4)`,
			updatedAddress, city, postalCode, phone)
		if err != nil {
			return fmt.Errorf("failed to insert service center %d: %v", i+1, err)
		}
	}
	return nil
}

func CreateEmployees(ctx context.Context, db *pgxpool.Pool, employeesCount int, users *[]*model.User) error {

	empRoles := []string{"Analyst", "Master", "Manager"}

	rows, err := db.Query(ctx, `SELECT service_center_id FROM service_centers`)
	if err != nil {
		log.Fatal("Failed to execute query:", err)
		return fmt.Errorf("no service centers found")
	}
	defer rows.Close()

	var serviceCenterIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			log.Fatal("Failed to scan row:", err)
		}
		serviceCenterIDs = append(serviceCenterIDs, id)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("Error during rows iteration:", err)
		return fmt.Errorf("error during service centers loading")
	}

	for employeeID := 1; employeeID <= employeesCount; employeeID++ {
		password := gofakeit.Password(true, true, true, false, false, 10)
		username := gofakeit.Username()
		_, err := db.Exec(ctx, `SELECT create_user(
				$1,
				$2,
				$3,
				$4,
				$5,
				$6,
				$7,
				$8
			);`, gofakeit.Name(), gofakeit.Number(0, 30), gofakeit.Number(18, 65), gofakeit.Price(30000, 150000), username, password, empRoles[rand.IntN(len(empRoles))], serviceCenterIDs[rand.IntN(len(serviceCenterIDs))])

		user := &model.User{
			Login:    username,
			Password: password,
		}
		*users = append(*users, user)
		if err != nil {
			return fmt.Errorf("failed to map employee %d to service center: %v", employeeID, err)
		}
	}

	return nil
}

func CreateCustomers(ctx context.Context, db *pgxpool.Pool, customersCount int, maxSpentMoney float64) error {

	for i := 0; i < customersCount; i++ {
		fullName := gofakeit.Name()
		phone := gofakeit.Phone()
		spentMoney := gofakeit.Price(0, maxSpentMoney)

		_, err := db.Exec(ctx, `INSERT INTO customers (full_name, phone_number, spent_money)
			VALUES ($1, $2, $3)`,
			fullName, phone, spentMoney)
		if err != nil {
			return fmt.Errorf("failed to insert customer %d: %v", i+1, err)
		}
	}

	_, err := db.Exec(ctx, `UPDATE customers SET spent_money = spent_money + 1.00`)
	if err != nil {
		log.Fatalf("Ошибка выполнения UPDATE: %v", err)
	}

	return err
}

func CreateServices(ctx context.Context, db *pgxpool.Pool, servicesCount int, servicePricePair *Pair[float64]) error {

	vehicleType := []string{"Car", "Moto"}

	services := []string{
		"Замена масла",
		"Ремонт двигателя",
		"Шиномонтаж",
		"Диагностика подвески",
		"Замена тормозных колодок",
		"Замена аккумулятора",
		"Ремонт коробки передач",
		"Полировка кузова",
		"Заправка кондиционера",
		"Компьютерная диагностика",
	}

	for i := 0; i < servicesCount; i++ {
		price := gofakeit.Price(servicePricePair.First, servicePricePair.Second)

		_, err := db.Exec(ctx, `INSERT INTO services (full_name, vehicle_type, price)
			VALUES ($1, $2, $3)`,
			services[rand.IntN(len(services))], vehicleType[rand.IntN(len(vehicleType))], price)
		if err != nil {
			return fmt.Errorf("failed to insert service %d: %v", i+1, err)
		}
	}

	return nil
}

func CreateStockpile(ctx context.Context, db *pgxpool.Pool) error {

	address := gofakeit.Address().Address
	postalCode := gofakeit.Zip()
	phone := gofakeit.Phone()

	_, err := db.Exec(ctx, `
		INSERT INTO stockpile (full_address, postal_code, phone_number)
		VALUES ($1, $2, $3)`,
		address, postalCode, phone)
	if err != nil {
		return fmt.Errorf("failed to insert stockpile: %v", err)
	}
	return err
}

func CreateSpareParts(ctx context.Context, db *pgxpool.Pool, sparePartsCount int, sparePartsPrice *Pair[float64], stockQuantity *Pair[int]) error {
	partNames := []string{
		"Фильтр масла", "Тормозной диск", "Стартер", "Свеча зажигания", "Шаровая опора",
		"Ремень ГРМ", "Топливный насос", "Амортизатор", "Подшипник ступицы", "Сальник двигателя",
		"Пыльник амортизатора", "Радиатор охлаждения", "Колодки тормозные", "Термостат", "Крыло переднее",
		"Фара передняя", "Фонарь задний", "Глушитель", "Сцепление", "Ремень привода",
		"Трос сцепления", "Цепь ГРМ", "Масляный насос", "Крышка клапанов", "Водяной насос",
		"Топливный фильтр", "Прокладка ГБЦ", "Рулевой наконечник", "Втулка стабилизатора", "Рычаг подвески",
		"Радиатор отопителя", "Генератор", "Клапан рециркуляции", "Клапан ЕГР", "Приводной вал",
		"Турбокомпрессор", "Прокладка коллектора", "Датчик давления масла", "Датчик температуры",
		"Шкив коленвала", "Шрус", "Ремень вентилятора", "Ремкомплект тормозов", "Диск сцепления",
	}
	for i := 0; i < sparePartsCount; i++ {
		articleNumber := gofakeit.Number(1, 200000)
		description := gofakeit.Sentence(10)
		price := gofakeit.Price(sparePartsPrice.First, sparePartsPrice.Second)
		stockQuantity := gofakeit.Number(stockQuantity.First, stockQuantity.Second)

		_, err := db.Exec(ctx, `
			INSERT INTO spare_parts (name, article_number, description, price, stock_quantity, stockpile_id)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			partNames[rand.IntN(len(partNames))], articleNumber, description, price, stockQuantity, 1)
		if err != nil {
			return fmt.Errorf("failed to insert spare part %d: %v", i+1, err)
		}
	}

	return nil
}

func CreateOrders(ctx context.Context, db *pgxpool.Pool, createOrdersTries int, purchasePrice *Pair[float64], sparePartsCountPerOrder int, serviceCountPerOrder int) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < createOrdersTries; i++ {
		var serviceCenterId int
		err := db.QueryRow(ctx, `
			SELECT service_center_id
			FROM employee_service_center
			WHERE employee_role IN ('Manager', 'Master')
			GROUP BY service_center_id
			HAVING COUNT(DISTINCT employee_role) > 1
			ORDER BY RANDOM()
			LIMIT 1;
		`).Scan(&serviceCenterId)

		if err != nil {
			log.Fatal("Error getting service center:", err)
			continue
		}

		joins := []utils.Join{
			{
				Type:      "INNER",
				Table:     "employee_service_center",
				Condition: "employees.employee_id = employee_service_center.employee_id",
			},
		}

		whereClause := squirrel.And{
			squirrel.Eq{"employee_service_center.employee_role": "Master"},
			squirrel.Eq{"employee_service_center.service_center_id": serviceCenterId},
		}

		masterId, err := utils.RandomIDWithBuilder(ctx, db, "employees", "employee_id",
			utils.WithJoins(joins), utils.WithWhereClause(whereClause))

		if err != nil {
			log.Println("Error getting master in service center", err)
			continue
		}

		whereClause = squirrel.And{
			squirrel.Eq{"employee_role": "Manager"},
			squirrel.Eq{"employee_service_center.service_center_id": serviceCenterId},
		}

		managerId, err := utils.RandomIDWithBuilder(ctx, db, "employees", "employee_id",
			utils.WithJoins(joins), utils.WithWhereClause(whereClause))
		if err != nil {
			log.Println("Error getting meneger in service center", err)
			continue
		}
		customerId, err := utils.RandomIDWithBuilder(ctx, db, "customers", "customer_id")
		if err != nil {
			log.Println("Error getting customer", err)
			continue
		}
		orderStatuses := []string{"Pending", "In Progress", "Completed"}
		var orderID int
		err = tx.QueryRow(ctx, `
			INSERT INTO orders
			(customer_id, service_center_id, manager_id, assigned_master_id, scheduled_date, status)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING order_id`,
			customerId, serviceCenterId, managerId, masterId, gofakeit.FutureDate(), orderStatuses[rand.IntN(len(orderStatuses))]).Scan(&orderID)

		if err != nil {
			return fmt.Errorf("failed to insert order %d: %v", i+1, err)
		}

		usedSpareParts := make(map[int]bool)
		for v := 0; v < sparePartsCountPerOrder; v++ {
			sparePartId, err := utils.RandomIDWithBuilder(ctx, db, "spare_parts", "part_id")
			if err != nil {
				log.Fatal("Error getting stockpile:", err)
				continue
			}

			if usedSpareParts[sparePartId] {
				continue
			}
			usedSpareParts[sparePartId] = true

			quantity := gofakeit.Number(1, 5)

			var stockQuantity int
			err = tx.QueryRow(ctx, "SELECT stock_quantity FROM spare_parts WHERE part_id = $1 FOR UPDATE", sparePartId).Scan(&stockQuantity)
			if err != nil {
				continue
			}

			if stockQuantity < quantity {
				log.Printf("Not enugh spare parts for (part_id: %d). Need: %d, On stockpile: %d. Skipping.\n", sparePartId, quantity, stockQuantity)
				continue
			}

			_, err = tx.Exec(ctx, `
				INSERT INTO spare_part_order (part_id, order_id, quantity, purchase_price)
				VALUES ($1, $2, $3, $4)`,
				sparePartId, orderID, quantity, gofakeit.Price(purchasePrice.First, purchasePrice.Second))
			if err != nil {
				return fmt.Errorf("failed to insert spare part %d: %v", i+1, err)
			}
		}

		usedService := make(map[int]bool)
		for v := 0; v < serviceCountPerOrder; v++ {
			serviceID, err := utils.RandomIDWithBuilder(ctx, db, "services", "service_id")
			if err != nil {
				log.Fatal("Error getting service:", err)
				continue
			}

			if usedService[serviceID] {
				continue
			}
			usedService[serviceID] = true

			_, err = tx.Exec(ctx, `
				INSERT INTO service_order (service_id, order_id)
				VALUES ($1, $2)`,
				serviceID, orderID)
			if err != nil {
				return fmt.Errorf("failed to insert spare part %d: %v", i+1, err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func InitAdmin(ctx context.Context, db *pgxpool.Pool) error {
	var count int
	err := db.QueryRow(ctx, `SELECT COUNT(*) FROM employee_service_center WHERE employee_role = $1`, "Administrator").Scan(&count)
	if err != nil {
		return fmt.Errorf("admin select err: %v", err)
	}

	if count > 0 {
		return fmt.Errorf("admin already exist")
	}
	currPath, err := os.Getwd()
	if err != nil {
		return err
	}
	file, err := os.Open(fmt.Sprintf("%s/internal/gomock/scripts/init_admin.sql", currPath))
	if err != nil {
		return fmt.Errorf("file open err: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("file read err: %w", err)
	}

	_, err = db.Exec(ctx, string(content))

	if err != nil {
		return fmt.Errorf("admin inserting err: %v", err)
	}
	return nil
}

func CreateReceipts(ctx context.Context, db *pgxpool.Pool) error {
	type receiptDTO struct {
		OrderId     int
		BonusPoints float64
		TotalCost   float64
	}
	var receiptDTOs []receiptDTO

	rows, err := db.Query(ctx, `
        SELECT o.order_id, o.total_cost, c.bonus_points
        FROM orders o
        JOIN customers c ON o.customer_id = c.customer_id
        LEFT JOIN receipts r ON o.order_id = r.order_id
        WHERE o.status = 'Completed' AND r.order_id IS NULL
    `)

	if err != nil {
		return fmt.Errorf("failed to query completed orders: %w", err)
	}

	for rows.Next() {
		var ro receiptDTO
		if err := rows.Scan(&ro.OrderId, &ro.TotalCost, &ro.BonusPoints); err != nil {
			return fmt.Errorf("failed to scan order: %w", err)
		}
		receiptDTOs = append(receiptDTOs, ro)
	}

	for _, receiptDTO := range receiptDTOs {
		var existingReceiptID int
		err = db.QueryRow(ctx, `
			SELECT receipt_id
			FROM receipts
			WHERE order_id = $1
			LIMIT 1
		`, receiptDTO.OrderId).Scan(&existingReceiptID)
		if err == nil {
			continue
		}

		spentBonusPoints := receiptDTO.BonusPoints * (rand.Float64()*0.9 + 0.1)
		_, err := db.Exec(ctx, `INSERT INTO receipts (order_id, bonus_points_spent, total_paid)
            VALUES ($1, $2, $3)`, receiptDTO.OrderId, spentBonusPoints, receiptDTO.TotalCost-spentBonusPoints)
		if err != nil {
			return fmt.Errorf("error insert receipt %w", err)
		}
	}
	return nil
}
