FROM golang:1.23

# Install PostgreSQL client
RUN apt-get update && apt-get install -y postgresql-client

WORKDIR /app
COPY . .

RUN go build -o main ./cmd/api

# Make scripts executable
RUN chmod +x ./scripts/db/setup_db.sh

# Create entrypoint script
RUN echo '#!/bin/bash\n\
    set -e\n\
    echo "Running database setup script..."\n\
    bash ./scripts/db/setup_db.sh\n\
    echo "Starting the application..."\n\
    exec ./main' > entrypoint.sh && chmod +x entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["./entrypoint.sh"]