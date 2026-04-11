CREATE TABLE fx_rates (
    fiat text NOT NULL,
    day date NOT NULL,
    rate numeric NOT NULL,
    is_real boolean NOT NULL DEFAULT true,
    source text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (fiat, day)
);

CREATE INDEX idx_fx_rates_fiat_day_desc ON fx_rates (fiat, day DESC);

CREATE INDEX idx_fx_rates_fiat_real_day_desc ON fx_rates (fiat, day DESC)
    WHERE is_real = true;
