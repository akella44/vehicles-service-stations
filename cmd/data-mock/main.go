package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"vehicles-service-stations/config"
	"vehicles-service-stations/internal/db"
	"vehicles-service-stations/internal/gomock"
	"vehicles-service-stations/internal/model"
)

var (
	skipAdminInit         bool
	skipEmployeesCreation bool
	employeesCount        int
	ordersCount           int
	customersCount        int
	serviceCentersCount   int
)

func main() {
	flag.BoolVar(&skipAdminInit, "skip-admin-init", false, "Skip initializing admin")
	flag.BoolVar(&skipEmployeesCreation, "skip-employees-creation", false, "Create additional employees")
	flag.IntVar(&employeesCount, "ec", 150, "Number of additional employees to create")
	flag.IntVar(&ordersCount, "oc", 200, "Number of additional orders to create")
	flag.IntVar(&customersCount, "cc", 100, "Number of additional customers to create")
	flag.IntVar(&serviceCentersCount, "sc", 50, "Number of additional service centers to create")
	flag.Parse()

	env_cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Ошибка создания конфигурации: %v", err)
		panic("error during cfg loading")
	}
	cfg, err := db.NewConfig(
		env_cfg,
		env_cfg.DbSuperuser,
		env_cfg.DbPassword,
	)
	if err != nil {
		log.Fatalf("Ошибка создания конфигурации: %v", err)
	}

	connManager := db.NewConnectionManager()
	connManager.AddPool(context.Background(), "superuser", cfg.ConnectionString())

	defer connManager.CloseAll()

	log.Println("Creating service centers...")
	if err := gomock.CreateServiceCenters(context.Background(), connManager.GetPool("superuser"), 3, serviceCentersCount); err != nil {
		log.Fatalf("Create service centers err %v", err)
	}
	log.Println("Creating service centers done")

	log.Println("Initing admin...")
	if !skipAdminInit {
		if err := gomock.InitAdmin(context.Background(), connManager.GetPool("superuser")); err != nil {
			log.Fatalf("Init admin err %v", err)
		}
	}
	log.Println("Init admin done")

	cfg, err = db.NewConfig(
		env_cfg,
		"admin_user",
		"StrongPassword123!",
	)
	if err != nil {
		log.Fatalf("Ошибка создания конфигурации: %v", err)
	}

	connManager.AddPool(context.Background(), "admin", cfg.ConnectionString())

	var wg sync.WaitGroup
	errCh := make(chan error, 5)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-ctx.Done():
			return
		default:
		}
		log.Println("Creating stockpile...")
		if err := gomock.CreateStockpile(ctx, connManager.GetPool("superuser")); err != nil {
			errCh <- err
		}
		log.Println("Creating stockpile done")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-ctx.Done():
			return
		default:
		}
		log.Println("Creating customers...")
		if err := gomock.CreateCustomers(ctx, connManager.GetPool("superuser"), customersCount, 150000.0); err != nil {
			errCh <- err
		}
		log.Println("Creating customers done")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-ctx.Done():
			return
		default:
		}
		log.Println("Creating services...")
		if err := gomock.CreateServices(ctx, connManager.GetPool("superuser"), 20, &gomock.Pair[float64]{First: 2000.0, Second: 50000.0}); err != nil {
			errCh <- err
		}
		log.Println("Creating services done")
	}()

	var users []*model.User

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-ctx.Done():
			return
		default:
		}
		log.Println("Creating employees...")
		if !skipEmployeesCreation {
			if err := gomock.CreateEmployees(ctx, connManager.GetPool("admin"), employeesCount, &users); err != nil {
				errCh <- err
			}
		}
		log.Println("Creating employees done")
		log.Println("Saving users cred to file...")
		file, err := os.Create("/tmp/creds.json")
		if err != nil {
			errCh <- fmt.Errorf("error while creation db users cred file")
		}
		encoder := json.NewEncoder(file)
		if err := encoder.Encode(users); err != nil {
			errCh <- fmt.Errorf("error while users creds serialization")
		}
		defer func() {
			if cerr := file.Close(); cerr != nil {
				errCh <- fmt.Errorf("error while file closing")
			}
		}()
		log.Println("Saving users cred to file done")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-ctx.Done():
			return
		default:
		}
		log.Println("Creating spare parts...")
		if err := gomock.CreateSpareParts(ctx, connManager.GetPool("superuser"), 30, &gomock.Pair[float64]{First: 500.0, Second: 500000.0}, &gomock.Pair[int]{First: 10, Second: 100}); err != nil {
			errCh <- err
		}
		log.Println("Creating spare parts done")
	}()

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		log.Fatalf("Err while executing mock funcs %v", err)
	}
	wg.Wait()
	log.Println("Creating orders...")
	if err := gomock.CreateOrders(ctx, connManager.GetPool("superuser"), ordersCount, &gomock.Pair[float64]{First: 500.0, Second: 500000.0}, 5, 2); err != nil {
		log.Fatalf("Err while executing creation of orders mock func %v", err)
	}
	log.Println("Creating orders done")
	log.Println("Creating receipts for completed orders...")
	if err := gomock.CreateReceipts(ctx, connManager.GetPool("superuser")); err != nil {
		log.Fatalf("Err while executing creation of receipts mock func %v", err)
	}
	log.Println("Creating receipts done")
	log.Println("No problem found, the end")
}
