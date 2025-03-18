Create a secrets/ Directory:

    mkdir -p secrets

Create Secret Files:

    echo "admin" > secrets/mongodb_user
    openssl rand -base64 12 > secrets/mongodb_password

Deploy MongoDB Container:

    docker-compose up -d
