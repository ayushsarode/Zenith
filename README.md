<p align="center">
  <img src="https://github.com/user-attachments/assets/808e7144-2147-4a41-afb7-dcea7f6dade5" width="1000px">
</p>




# Zenith
A terminal based chat application built in go with gRPC

# Application Architecture

![diagram-export-3-20-2025-11_02_24-PM](https://github.com/user-attachments/assets/49fae19a-e86f-41bd-b0d5-38f7c8d35aa8)


## Prerequisites

- [Go](https://golang.org/doc/install) (version 1.24 or higher)
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)


## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/ayushsarode/zenith.git
   cd zenith
   ```

2. Create a `.env` file in the project root:
   ```bash
    DB_HOST=your_db_host
    DB_PORT=5432
    DB_USER=your_db_user
    DB_PASSWORD=your_db_password
    DB_NAME=your_db_name
    DB_SSLMODE=disable

    SERVER_HOST=server
    SERVER_PORT=50051

    POSTGRES_USER=your_postgres_user
    POSTGRES_PASSWORD=your_postgres_password
    POSTGRES_DB=your_postgres_db
   ```

## 🛠 Usage

### Start the Server

```bash
make server
```

### Start the Client to Chat

```bash
make client
```

## 📋 Other Commands

To view all available commands:

```bash
make help
```


## 🤝 Contributing

Contributions and suggestions are welcome — feel free to open an issue or pull request!




