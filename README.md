# LoupeBox

# Dependencies

## Postgres

```
sudo su postgres
psql
CREATE DATABASE loupebox;
```

You'll also need to expore the database url in your config:

```
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/loupebox

```

## Darktable

Loupebox uses Darktable to generate thumbnails. To install Darktable on Ubuntu run

```
sudo apt-get install darktable
```

## Quick Start

Initialize new collection

```
loupebox init
```

