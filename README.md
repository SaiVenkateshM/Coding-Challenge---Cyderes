# Coding-Challenge---Cyderes
# Data Ingestion Service

A robust, cloud-native data ingestion service built in Go that collects data from external APIs, processes it, and stores it in scalable cloud storage solutions.

## Features

- **Multi-Storage Support**: DynamoDB, MongoDB, PostgreSQL
- **Retry Logic**: Configurable retry mechanisms with exponential backoff
- **RESTful API**: Retrieve ingested data via HTTP endpoints
- **Health Monitoring**: Built-in health checks and ingestion status tracking
- **Containerized**: Full Docker support with docker-compose
- **CI/CD Ready**: GitHub Actions workflow for automated testing and deployment
- **Graceful Shutdown**: Proper cleanup on termination signals

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   External API  │    │  Ingestion      │    │  Cloud Storage  │
│                 │◄──►│  Service        │◄──►│  (DynamoDB/     │
│  JSONPlaceholder│    │                 │    │   MongoDB/      │
└─────────────────┘    └─────────────────┘    │   PostgreSQL)   │
                                │              └─────────────────┘
                                ▼              
                       ┌─────────────────┐    
                       │   HTTP Server   │    
                       │  (REST API)     │    
                       └─────────────────┘    
```

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- AWS CLI (for DynamoDB deployment)

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-org/data-ingestion-service.git
   cd data-ingestion-service
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Run with Docker Compose**
   ```bash
   docker-compose up --build
   ```

4. **Test the service**
   ```bash
   # Health check
   curl http://localhost:8080/health
   
   # Get ingested posts
   curl http://localhost:8080/posts
   
   # Get specific post
   curl http://localhost:8080/posts/1
   
   # Check ingestion status
   curl http://localhost:8080/status
   ```

## Configuration

Configure the service using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `STORAGE_TYPE` | Storage backend (dynamodb/mongodb/postgresql) | `dynamodb` |
| `AWS_REGION` | AWS region for DynamoDB | `us-west-2` |
| `TABLE_NAME` | Storage table name | `ingested_data` |
| `DYNAMODB_ENDPOINT` | DynamoDB endpoint (for local testing) | `` |
| `MONGODB_URI` | MongoDB connection string | `` |
| `POSTGRES_URI` | PostgreSQL connection string | `` |
| `API_ENDPOINT` | External API endpoint | `https://jsonplaceholder.typicode.com/posts` |
| `INGESTION_INTERVAL` | How often to fetch data | `5m` |
| `API_TIMEOUT` | API request timeout | `30s` |
| `RETRY_COUNT` | Number of retry attempts | `3` |
| `SERVER_PORT` | HTTP server port | `8080` |

## Storage Options

### DynamoDB (Recommended)

**Why DynamoDB?**
- **Serverless**: No infrastructure management
- **Scalability**: Auto-scales based on demand
- **Performance**: Single-digit millisecond latency
- **Cost-effective**: Pay-per-request pricing model
- **Reliability**: 99.999% availability SLA

**Trade-offs:**
- AWS vendor lock-in
- Query limitations compared to SQL databases
- Learning curve for NoSQL concepts

### MongoDB

**Setup:**
```bash
# Update docker-compose.yml to use MongoDB
STORAGE_TYPE=mongodb
MONGODB_URI=mongodb://admin:password@mongodb:27017/ingestion_db?authSource=admin
```

### PostgreSQL

**Setup:**
```bash
# Update docker-compose.yml to use PostgreSQL
STORAGE_TYPE=postgresql
POSTGRES_URI=postgres://admin:password@postgres:5432/ingestion_db?sslmode=disable
```

## API Endpoints

### GET /health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "time": "2024-01-15T10:30:00Z"
}
```

### GET /posts
Retrieve ingested posts with pagination.

**Query Parameters:**
- `limit` (int): Number of posts to return (default: 10)
- `offset` (int): Number of posts to skip (default: 0)

**Response:**
```json
{
  "posts": [
    {
      "userId": 1,
      "id": 1,
      "title": "Post Title",
      "body": "Post content...",
      "ingested_at": "2024-01-15T10:30:00Z",
      "source": "placeholder_api"
    }
  ],
  "count": 1,
  "limit": 10,
  "offset": 0
}
```

### GET /posts/{id}
Retrieve a specific post by ID.

**Response:**
```json
{
  "userId": 1,
  "id": 1,
  "title": "Post Title",
  "body": "Post content...",
  "ingested_at": "2024-01-15T10:30:00Z",
  "source": "placeholder_api"
}
```

### GET /status
Get ingestion status and statistics.

**Response:**
```json
{
  "last_successful_run": "2024-01-15T10:30:00Z",
  "last_attempt": "2024-01-15T10:30:00Z",
  "status": "success",
  "records_ingested": 100
}
```

## Testing

### Unit Tests
```bash
go test -v ./...
```

### Integration Tests
```bash
# Start dependencies
docker-compose up -d dynamodb

# Run integration tests
go test -v -tags=integration ./tests/integration/...
```

### Test Coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Deployment

### AWS ECS Deployment

1. **Create ECS Task Definition**
   ```json
   {
     "family": "data-ingestion-task",
     "networkMode": "awsvpc",
     "requiresCompatibilities": ["FARGATE"],
     "cpu": "256",
     "memory": "512",
     "containerDefinitions": [
       {
         "name": "data-ingestion-service",
         "image": "your-registry/data-ingestion-service:latest",
         "essential": true,
         "portMappings": [
           {
             "containerPort": 8080,
             "protocol": "tcp"
           }
         ],
         "environment": [
           {
             "name": "STORAGE_TYPE",
             "value": "dynamodb"
           },
           {
             "name": "AWS_REGION",
             "value": "us-west-2"
           }
         ]
       }
     ]
   }
   ```

2. **Create ECS Service**
   ```bash
   aws ecs create-service \
     --cluster data-ingestion-cluster \
     --service-name data-ingestion-service \
     --task-definition data-ingestion-task \
     --desired-count 2 \
     --launch-type FARGATE
   ```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: data-ingestion-service
spec:
  replicas: 2
  selector:
    matchLabels:
      app: data-ingestion-service
  template:
    metadata:
      labels:
        app: data-ingestion-service
    spec:
      containers:
      - name: data-ingestion-service
        image: your-registry/data-ingestion-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: STORAGE_TYPE
          value: "dynamodb"
        - name: AWS_REGION
          value: "us-west-2"
```

## Monitoring and Observability

### Health Checks
The service provides built-in health checks at `/health` endpoint.

### Metrics
Consider integrating with:
- **Prometheus**: For metrics collection
- **Grafana**: For metrics visualization
- **AWS CloudWatch**: For AWS-native monitoring

### Logging
The service uses structured logging. Configure log levels and formats based on your environment.

## Development Notes

### Hardest Parts to Implement

1. **Retry Logic with Exponential Backoff**: Implementing robust retry mechanisms while avoiding thundering herd problems.

2. **Graceful Shutdown**: Ensuring all goroutines properly cleanup and data consistency during shutdown.

3. **Storage Abstraction**: Creating a flexible interface that works across different storage backends while maintaining performance.

4. **Error Handling**: Comprehensive error handling that provides meaningful feedback without exposing internal details.

### Trade-offs Considered

1. **Storage Choice**: 
   - DynamoDB: High performance, scalability vs. vendor lock-in
   - MongoDB: Flexibility vs. operational complexity
   - PostgreSQL: ACID compliance vs. scaling challenges

2. **Ingestion Strategy**:
   - Pull-based (current): Simple, reliable vs. potential delays
   - Push-based: Real-time vs. complexity, security concerns
   - Event-driven: Highly scalable vs. infrastructure overhead

3. **Data Consistency**:
   - Eventually consistent (DynamoDB): Performance vs. consistency
   - Strong consistency: Reliability vs. performance

### Future Improvements

1. **Distributed Processing**: Implement worker pools for parallel ingestion
2. **Data Validation**: Add schema validation for ingested data
3. **Caching Layer**: Redis/ElastiCache for frequently accessed data
4. **Monitoring**: Enhanced metrics and alerting
5. **Data Transformation**: More sophisticated ETL capabilities
6. **API Rate Limiting**: Protect against abuse
7. **Authentication**: Add API key or OAuth support
8. **Data Archiving**: Automatic archival of old data
9. **Multi-region Support**: Cross-region replication
10. **Stream Processing**: Real-time data processing with Kafka/Kinesis

## Tracking Latest Successful Ingestion

### Approach
The service implements ingestion tracking through:

1. **Status Records**: Store ingestion metadata in a separate table/collection
2. **Atomic Updates**: Update status atomically with data ingestion
3. **Timestamp Tracking**: Track both attempt and success timestamps
4. **Error Logging**: Capture and store error details for debugging

### Challenges and Trade-offs

**Challenges:**
- **Consistency**: Ensuring status updates are consistent with data ingestion
- **Performance**: Minimizing overhead of status tracking
- **Storage**: Managing status data growth over time

**Trade-offs:**
- **Accuracy vs Performance**: More frequent status updates vs. system overhead
- **Storage vs Complexity**: Separate status storage vs. inline metadata
- **Granularity vs Simplicity**: Per-record vs. batch-level tracking

###