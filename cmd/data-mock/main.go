package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"vehicles-service-stations/internal/db"
	"vehicles-service-stations/internal/gomock"
	"vehicles-service-stations/internal/model"
)

const (
	dbIP   = "192.168.1.62"
	dbPort = 5432
	dbName = "edu"
)

func main() {
	cfg, err := db.NewConfig(
		dbIP,
		dbPort,
		"arklim",
		"qwerty",
		dbName,
	)
	if err != nil {
		log.Fatalf("Ошибка создания конфигурации: %v", err)
	}

	connManager := db.NewConnectionManager()
	if err != nil {
		log.Fatalf("Ошибка создания менеджера соединений: %v", err)
	}

	connManager.AddPool(context.Background(), "superuser", cfg.ConnectionString())

	defer connManager.CloseAll()

	log.Println("Creating service centers...")
	if err := gomock.CreateServiceCenters(context.Background(), connManager.GetPool("superuser"), 3, 10); err != nil {
		log.Fatalf("Create service centers err %v", err)
	}
	log.Println("Creating service centers done")

	log.Println("Initing admin...")
	if err := gomock.InitAdmin(context.Background(), connManager.GetPool("superuser")); err != nil {
		log.Fatalf("Init admin err %v", err)
	}
	log.Println("Init admin done")

	cfg, err = db.NewConfig(
		dbIP,
		dbPort,
		"admin_user",
		"StrongPassword123!",
		dbName,
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
		if err := gomock.CreateCustomers(ctx, connManager.GetPool("superuser"), 100, 150000.0); err != nil {
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
		if err := gomock.CreateEmployees(ctx, connManager.GetPool("admin"), 150, &users); err != nil {
			errCh <- err
		}
		log.Println("Creating employees done")
		log.Println("Saving users cred to file...")
		log.Println(users)
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
	if err := gomock.CreateOrders(ctx, connManager.GetPool("superuser"), 200, &gomock.Pair[float64]{First: 500.0, Second: 500000.0}, 5, 2); err != nil {
		log.Fatalf("Err while executing mock func %v", err)
	}
	log.Println("Creating orders done")
	log.Println("No problem found, the end")
}
