CREATE TABLE airports (
    id                SERIAL          PRIMARY KEY,
    iata_code         VARCHAR(3)      NOT NULL UNIQUE,
    name              TEXT            NOT NULL,
    city              VARCHAR(100)    NOT NULL,
    country           VARCHAR(100)    NOT NULL,
    opened_date       DATE            NOT NULL,
    runways_count     INTEGER         NOT NULL CHECK (runways_count > 0),
    runway_length_km NUMERIC(5, 2)   NOT NULL CHECK (runway_length_km > 0),
    altitude_m        INTEGER         NOT NULL DEFAULT 0
);

CREATE TABLE flights (
    id                SERIAL          PRIMARY KEY,
    flight_number     VARCHAR(10)     NOT NULL UNIQUE,
    airport_from_id   INTEGER         NOT NULL,
    airport_to_id     INTEGER         NOT NULL,
    departure_date    DATE            NOT NULL,
    duration_minutes INTEGER         NOT NULL CHECK (duration_minutes > 0),
    distance_km       NUMERIC(7, 2)   NOT NULL CHECK (distance_km > 0),
    base_price_usd    NUMERIC(10, 2)  NOT NULL CHECK (base_price_usd >= 0),
    seats_total       INTEGER         NOT NULL CHECK (seats_total > 0),

    CONSTRAINT fk_airport_from
        FOREIGN KEY (airport_from_id)
        REFERENCES airports (id)
        ON DELETE RESTRICT
        ON UPDATE CASCADE,

    CONSTRAINT fk_airport_to
        FOREIGN KEY (airport_to_id)
        REFERENCES airports (id)
        ON DELETE RESTRICT
        ON UPDATE CASCADE,

    CONSTRAINT chk_different_airports
        CHECK (airport_from_id <> airport_to_id)
);