#!/bin/bash

# Create test database
PGPASSWORD=postgres psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS alchemorsel_test;"
PGPASSWORD=postgres psql -h localhost -U postgres -c "CREATE DATABASE alchemorsel_test;" 