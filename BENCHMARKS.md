# Результаты нагрузочного тестирования (бенчмарки)

## Окружение
* **OS:** Linux (amd64)
* **CPU:** 12th Gen Intel(R) Core(TM) i5-12450H (4 логических ядра выделено под тесты)
* **Go Version:** 1.26.1

---

## Микро-бенчмарки ядра

Тестирование структуры `RingBuffer` (In-Memory). Для оценки стабильности архитектуры и влияния сборщика мусора на больших объемах памяти, тесты были проведены на датасетах различного размера. Для экстремальных объемов (50 млн) использовалась переменная окружения `GOMEMLIMIT=12GiB`, чтобы ограничить потребление оперативный памяти и предотвратить использование файлов подкачки.

### Сводная таблица масштабирования (Scaling)

| Объем датасета | Метод | Конкурентность | Время (ns/op) | Память (B/op) | Аллокации (allocs/op) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **1 000 000** | `AddSearchEvent` | 1 ядро | ~505.9 ns/op | ~265 B/op | **0** |
| | `AddSearchEvent_Parallel` | 4 ядра | ~203.5 ns/op | ~91 B/op | 3* |
| | `GetTopNTrends` | 1 ядро | ~13.99 ns/op | **0 B/op** | **0** |
| **10 000 000** | `AddSearchEvent` | 1 ядро | ~491.5 ns/op | ~263 B/op | **0** |
| | `AddSearchEvent_Parallel` | 4 ядра | ~319.9 ns/op | ~207 B/op | 3* |
| | `GetTopNTrends` | 1 ядро | ~14.83 ns/op | **0 B/op** | **0** |
| **50 000 000** | `AddSearchEvent` | 1 ядро | ~513.0 ns/op | ~267 B/op | **0** |
| | `AddSearchEvent_Parallel` | 4 ядра | ~287.0 ns/op | ~206 B/op | 3* |
| | `GetTopNTrends` | 1 ядро | ~15.63 ns/op | **0 B/op** | **0** |

\**(Примечание: 3 аллокации в параллельном тесте связаны с генерацией строк `fmt.Sprintf` внутри цикла самого бенчмарка. Сам алгоритм буфера работает без аллокаций).*

## E2E нагрузочное тестирование (gRPC)

Тестирование транспортного слоя с помощью утилиты [ghz](https://github.com/bojand/ghz). Перед началом обстрела в In-Memory ядро через Kafka-сидер было загружено 1 000 000 событий (10 000 уникальных запросов).

### Тест 1: базовая нагрузка (100k запросов, 128 потоков)

#### Команда:
```bash
ghz --insecure --proto=api/trend/v1/trend.proto --call=trendservice.TrendService.GetTopN --data='{"limit": 10}' -n 100000 -c 128 127.0.0.1:50051
```

#### Отчет о тестировании:

```plaintext
Summary:
  Count:	100000
  Total:	2.16 s
  Slowest:	11.72 ms
  Fastest:	0.21 ms
  Average:	2.07 ms
  Requests/sec:	46306.59

Latency distribution:
  10 % in 1.02 ms 
  25 % in 1.38 ms 
  50 % in 1.91 ms 
  75 % in 2.53 ms 
  90 % in 3.28 ms 
  95 % in 3.89 ms 
  99 % in 5.40 ms 

Status code distribution:
  [OK]   100000 responses
```

### Тест 2: высокая нагрузка (1M запросов, 256 потоков)

#### Команда:
```bash
ghz --insecure --proto=api/trend/v1/trend.proto --call=trendservice.TrendService.GetTopN --data='{"limit": 10}' -n 1000000 -c 256 127.0.0.1:50051
```

#### Отчет о тестировании:

```plaintext
Summary:
  Count:	1000000
  Total:	21.51 s
  Slowest:	38.55 ms
  Fastest:	0.16 ms
  Average:	4.22 ms
  Requests/sec:	46491.51

Latency distribution:
  10 % in 2.38 ms 
  25 % in 2.96 ms 
  50 % in 3.75 ms 
  75 % in 5.04 ms 
  90 % in 6.66 ms 
  95 % in 7.76 ms 
  99 % in 10.87 ms 

Status code distribution:
  [OK]   1000000 responses
```

### Тест 2: экстремальная нагрузка (10M запросов, 512 потоков)

#### Команда:
```bash
ghz --insecure --proto=api/trend/v1/trend.proto --call=trendservice.TrendService.GetTopN --data='{"limit": 10}' -n 10000000 -c 512 127.0.0.1:50051
```

#### Отчет о тестировании:

```plaintext
Summary:
  Count:	10000000
  Total:	209.07 s
  Slowest:	53.39 ms
  Fastest:	0.24 ms
  Average:	8.12 ms
  Requests/sec:	47830.73

Latency distribution:
  10 % in 4.76 ms 
  25 % in 5.98 ms 
  50 % in 7.71 ms 
  75 % in 10.29 ms 
  90 % in 13.11 ms 
  95 % in 15.12 ms 
  99 % in 20.31 ms 

Status code distribution:
  [OK]   10000000 responses
```
