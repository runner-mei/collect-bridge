CREATE TABLE IF NOT EXISTS tpt_alert_cookies (
  id                SERIAL  PRIMARY KEY,
  rule_id           bigint,
  managed_type      varchar(200),
  managed_id        bigint,
  status            int,
  current_value     varchar(2000),
  triggered_at      timestamp with time zone,

  CONSTRAINT tpt_alert_cookies_unique_rule_id UNIQUE (rule_id)
);
DROP INDEX IF EXISTS tpt_alert_cookies_rule_id_idx;
CREATE INDEX tpt_alert_cookies_rule_id_idx ON tpt_alert_cookies USING btree (rule_id);

CREATE TABLE IF NOT EXISTS tpt_alert_histories (
  id                SERIAL  PRIMARY KEY,
  rule_id           bigint,
  managed_type      varchar(200),
  managed_id        bigint,
  status            int,
  current_value     varchar(2000),
  triggered_at      timestamp with time zone
);
DROP INDEX IF EXISTS tpt_alert_histories_mo_id_idx;
CREATE INDEX tpt_alert_histories_mo_id_idx ON tpt_alert_histories USING btree (managed_type COLLATE pg_catalog."default", managed_id);
DROP INDEX IF EXISTS tpt_alert_histories_rule_id_idx;
CREATE INDEX tpt_alert_histories_rule_id_idx ON tpt_alert_histories USING btree (rule_id);

CREATE OR REPLACE FUNCTION tpt_alert_histories_partition_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
  index_query text;
BEGIN
  FOR create_query, index_query IN SELECT
      'CREATE TABLE IF NOT EXISTS tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( CHECK( triggered_at >= timestamp '''
      || TO_CHAR( d, 'YYYY-MM-DD 00:00:00' )
      || ''' AND triggered_at < timestamp '''
      || TO_CHAR( d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00' )
      || ''' ) ) inherits ( tpt_alert_histories );',
      'DROP INDEX  IF EXISTS tpt_alert_histories_'
      ||TO_CHAR( d, 'YYYY_MM' )
      ||'_triggered_at_idx;',
      'CREATE INDEX tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_triggered_at_idx ON tpt_alert_histories_' 
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( triggered_at );'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
    EXECUTE index_query;
  END LOOP;
END;
$$
language plpgsql;


CREATE OR REPLACE FUNCTION tpt_alert_histories_partition_deletion( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
  index_query text;
BEGIN
  FOR create_query, index_query IN SELECT
      'DROP INDEX IF EXISTS tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_rule_id_idx;',
      'DROP INDEX IF EXISTS tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_mo_id_idx;',
      'DROP INDEX  IF EXISTS tpt_alert_histories_'
      ||TO_CHAR( d, 'YYYY_MM' )
      ||'_triggered_at_idx;',
      'DROP INDEX  IF EXISTS tpt_alert_histories_'
      ||TO_CHAR( d, 'YYYY_MM' )
      ||'_triggered_at_idx;',
      'DROP TABLE IF EXISTS tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
    EXECUTE index_query;
  END LOOP;
END;
$$
language plpgsql;


CREATE OR REPLACE FUNCTION tpt_alert_histories_partition_triggered_at_idx_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
  index_query text;
BEGIN
  FOR create_query, index_query IN SELECT
      'DROP INDEX  IF EXISTS tpt_alert_histories_'
      ||TO_CHAR( d, 'YYYY_MM' )
      ||'_triggered_at_idx;',
      'CREATE INDEX tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_triggered_at_idx ON tpt_alert_histories_' 
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( triggered_at );'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
    EXECUTE index_query;
  END LOOP;
END;
$$
language plpgsql;


CREATE OR REPLACE FUNCTION tpt_alert_histories_partition_mo_id_idx_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
  index_query text;
BEGIN
  FOR create_query, index_query IN SELECT
       'DROP INDEX IF EXISTS tpt_alert_histories_'
       || TO_CHAR( d, 'YYYY_MM' )
       || '_mo_id_idx;',
      'CREATE INDEX tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_mo_id_idx ON tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' USING btree (managed_type COLLATE pg_catalog."default", managed_id);'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
    EXECUTE index_query;
  END LOOP;
END;
$$
language plpgsql;


CREATE OR REPLACE FUNCTION tpt_alert_histories_partition_rule_id_idx_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
  index_query text;
BEGIN
  FOR create_query, index_query IN SELECT
      'DROP INDEX IF EXISTS tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_rule_id_idx;',
      'CREATE INDEX tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_rule_id_idx ON tpt_alert_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' USING btree (rule_id);'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
    EXECUTE index_query;
  END LOOP;
END;
$$
language plpgsql;

SELECT tpt_alert_histories_partition_creation( '2010-01-01', '2028-01-01' );
SELECT tpt_alert_histories_partition_triggered_at_idx_creation( '2010-01-01', '2028-01-01' );
SELECT tpt_alert_histories_partition_mo_id_idx_creation( '2010-01-01', '2028-01-01' );
SELECT tpt_alert_histories_partition_rule_id_idx_creation( '2010-01-01', '2028-01-01' );

-- drop function tpt_alert_histories_partition_function();
CREATE OR REPLACE FUNCTION  tpt_alert_histories_partition_function()
returns TRIGGER AS $$
begin
  execute 'INSERT INTO tpt_alert_histories_'
    || to_char( NEW.triggered_at, 'YYYY_MM' )
    || '(rule_id, managed_type, managed_id, status, current_value, triggered_at) VALUES ($1, $2, $3, $4, $5, $6)' USING NEW.rule_id, NEW.managed_type, NEW.managed_id, NEW.status, NEW.current_value, NEW.triggered_at ;
  RETURN NULL;
end;
$$
LANGUAGE plpgsql;

-- drop trigger tpt_alert_histories_partition_trigger;
DROP TRIGGER IF EXISTS  tpt_alert_histories_partition_trigger ON tpt_alert_histories;
CREATE TRIGGER tpt_alert_histories_partition_trigger
  before INSERT
  ON tpt_alert_histories
  FOR each row
  execute procedure tpt_alert_histories_partition_function() ;

-- histories
CREATE TABLE IF NOT EXISTS tpt_histories (
  id                SERIAL  PRIMARY KEY,
  rule_id           bigint,
  managed_type      varchar(200),
  managed_id        bigint,
  current_value     NUMERIC(20, 4),
  sampled_at        timestamp with time zone
);
DROP INDEX IF EXISTS tpt_histories_mo_id_idx;
CREATE INDEX tpt_histories_mo_id_idx ON tpt_histories USING btree (managed_type COLLATE pg_catalog."default", managed_id);
DROP INDEX IF EXISTS tpt_histories_rule_id_idx;
CREATE INDEX tpt_histories_rule_id_idx ON tpt_histories USING btree (rule_id);

CREATE OR REPLACE FUNCTION tpt_histories_partition_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
  index_query text;
BEGIN
  FOR create_query, index_query IN SELECT
      'CREATE TABLE IF NOT EXISTS tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( CHECK( sampled_at >= timestamp '''
      || TO_CHAR( d, 'YYYY-MM-DD 00:00:00' )
      || ''' AND sampled_at < timestamp '''
      || TO_CHAR( d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00' )
      || ''' ) ) inherits ( tpt_histories );',
      'DROP INDEX  IF EXISTS tpt_histories_'
      ||TO_CHAR( d, 'YYYY_MM' )
      ||'_sampled_at_idx;',
      'CREATE INDEX tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_sampled_at_idx ON tpt_histories_' 
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( sampled_at );'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
    EXECUTE index_query;
  END LOOP;
END;
$$
language plpgsql;



CREATE OR REPLACE FUNCTION tpt_histories_partition_deletion( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
  index_query text;
BEGIN
  FOR create_query, index_query IN SELECT
      'DROP INDEX IF EXISTS tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_rule_id_idx;',
      'DROP INDEX IF EXISTS tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_mo_id_idx;',
      'DROP INDEX  IF EXISTS tpt_histories_'
      ||TO_CHAR( d, 'YYYY_MM' )
      ||'_sampled_at_idx;',
      'DROP TABLE IF EXISTS tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
    EXECUTE index_query;
  END LOOP;
END;
$$
language plpgsql;


CREATE OR REPLACE FUNCTION tpt_histories_partition_sampled_at_idx_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
  index_query text;
BEGIN
  FOR create_query, index_query IN SELECT
      'DROP INDEX  IF EXISTS tpt_histories_'
      ||TO_CHAR( d, 'YYYY_MM' )
      ||'_sampled_at_idx;',
      'CREATE INDEX tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_sampled_at_idx ON tpt_histories_' 
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( sampled_at );'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
    EXECUTE index_query;
  END LOOP;
END;
$$
language plpgsql;

CREATE OR REPLACE FUNCTION tpt_histories_partition_mo_id_idx_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
  index_query text;
BEGIN
  FOR create_query, index_query IN SELECT
       'DROP INDEX IF EXISTS tpt_histories_'
       || TO_CHAR( d, 'YYYY_MM' )
       || '_mo_id_idx;',
      'CREATE INDEX tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_mo_id_idx ON tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' USING btree (managed_type COLLATE pg_catalog."default", managed_id);'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
    EXECUTE index_query;
  END LOOP;
END;
$$
language plpgsql;

CREATE OR REPLACE FUNCTION tpt_histories_partition_rule_id_idx_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
  index_query text;
BEGIN
  FOR create_query, index_query IN SELECT
      'DROP INDEX IF EXISTS tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_rule_id_idx;',
      'CREATE INDEX tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || '_rule_id_idx ON tpt_histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' USING btree (rule_id);'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
    EXECUTE index_query;
  END LOOP;
END;
$$
language plpgsql;

SELECT tpt_histories_partition_creation( '2010-01-01', '2028-01-01' );
SELECT tpt_histories_partition_sampled_at_idx_creation( '2010-01-01', '2028-01-01' );
SELECT tpt_histories_partition_mo_id_idx_creation( '2010-01-01', '2028-01-01' );
SELECT tpt_histories_partition_rule_id_idx_creation( '2010-01-01', '2028-01-01' );

-- drop function tpt_histories_partition_function();
CREATE OR REPLACE FUNCTION  tpt_histories_partition_function()
returns TRIGGER AS $$
begin
  execute 'INSERT INTO tpt_histories_'
    || to_char( NEW.sampled_at, 'YYYY_MM' )
    || '(rule_id, managed_type, managed_id, current_value, sampled_at) VALUES ($1, $2, $3, $4, $5)' USING NEW.rule_id, NEW.managed_type, NEW.managed_id, NEW.current_value, NEW.sampled_at ;
  RETURN NULL;
end;
$$
LANGUAGE plpgsql;

-- drop trigger tpt_histories_partition_trigger;
DROP TRIGGER IF EXISTS  tpt_histories_partition_trigger ON tpt_histories;
CREATE TRIGGER tpt_histories_partition_trigger
  before INSERT
  ON tpt_histories
  FOR each row
  execute procedure tpt_histories_partition_function() ;




-- -- alert
-- DROP INDEX IF EXISTS tpt_alert_cookies_rule_id_idx;
-- DROP TABLE IF EXISTS tpt_alert_cookies CASCADE;

-- DROP INDEX IF EXISTS tpt_alert_histories_mo_id_idx;
-- DROP INDEX IF EXISTS tpt_alert_histories_rule_id_idx;
-- DROP TABLE IF EXISTS tpt_alert_histories CASCADE;

-- -- histories
-- DROP INDEX IF EXISTS tpt_histories_mo_id_idx;
-- DROP INDEX IF EXISTS tpt_histories_rule_id_idx;
-- DROP TABLE IF EXISTS tpt_histories CASCADE;


-- CREATE OR REPLACE FUNCTION tpt_alert_histories_insert_trigger()                      
-- RETURNS TRIGGER AS $$  
-- BEGIN  
--     IF ( NEW.triggered_at >= timestamp '2013-01-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-02-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m01 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-02-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-03-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m02 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-03-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-04-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m03 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-04-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-05-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m04 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-05-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-06-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m05 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-06-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-07-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m06 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-07-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-08-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m07 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-08-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-09-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m08 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-09-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-10-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m09 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-10-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-11-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m10 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-11-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2013-12-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m11 VALUES (NEW.*);  
--     ELSIF ( NEW.triggered_at >= timestamp '2013-12-01 00:00:00' AND    
--          NEW.triggered_at < timestamp '2014-01-01 00:00:00' ) THEN  
--         INSERT INTO tpt_alert_histories_y2013m12 VALUES (NEW.*);  
--     ELSE  
--         RAISE EXCEPTION 'timestamp out of range. Fix the tpt_alert_histories_insert_trigger() function!';  
--     END IF;  
--     RETURN NULL;  
-- END;  
-- $$  
-- LANGUAGE plpgsql;

-- DROP TRIGGER IF EXISTS  insert_tpt_alert_histories_trigger ON tpt_alert_histories;

-- CREATE TRIGGER insert_tpt_alert_histories_trigger
--    BEFORE INSERT ON tpt_alert_histories
--    FOR EACH ROW EXECUTE PROCEDURE tpt_alert_histories_insert_trigger();