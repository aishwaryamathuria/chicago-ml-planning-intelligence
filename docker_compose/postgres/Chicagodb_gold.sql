--
-- PostgreSQL database dump
--

-- Dumped from database version 14.15 (Postgres.app)
-- Dumped by pg_dump version 17.0

-- Started on 2025-06-07 21:50:24 IST

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

DROP DATABASE IF EXISTS "ChicagoBusinessIntelligence_GOLD";
--
-- TOC entry 3590 (class 1262 OID 45008)
-- Name: ChicagoBusinessIntelligence_GOLD; Type: DATABASE; Schema: -; Owner: postgres
--

CREATE DATABASE "ChicagoBusinessIntelligence_GOLD" WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.UTF-8';


ALTER DATABASE "ChicagoBusinessIntelligence_GOLD" OWNER TO postgres;

\connect "ChicagoBusinessIntelligence_GOLD"

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 4 (class 2615 OID 2200)
-- Name: public; Type: SCHEMA; Schema: -; Owner: postgres
--

-- *not* creating schema, since initdb creates it


ALTER SCHEMA public OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 212 (class 1259 OID 46471)
-- Name: building_permits; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.building_permits (
    record_id text,
    permit_type text,
    location text,
    community_area bigint,
    permit_number text,
    neighborhood text,
    permit_status text,
    zip_code double precision
);


ALTER TABLE public.building_permits OWNER TO postgres;

--
-- TOC entry 209 (class 1259 OID 46443)
-- Name: ccvi_index; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.ccvi_index (
    zip_code integer,
    ccvi_score double precision,
    ccvi_category character varying
);


ALTER TABLE public.ccvi_index OWNER TO postgres;

--
-- TOC entry 210 (class 1259 OID 46448)
-- Name: covid_cases; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.covid_cases (
    zip_code integer,
    start_date date,
    end_date date,
    case_count integer,
    positive_test_percent double precision
);


ALTER TABLE public.covid_cases OWNER TO postgres;

--
-- TOC entry 211 (class 1259 OID 46451)
-- Name: public_health_stats; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.public_health_stats (
    community_area integer NOT NULL,
    community_area_name text,
    below_poverty_level double precision,
    per_capita_income integer,
    unemployment double precision
);


ALTER TABLE public.public_health_stats OWNER TO postgres;

--
-- TOC entry 213 (class 1259 OID 46488)
-- Name: taxi_trips; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.taxi_trips (
    trip_id text,
    trip_start_timestamp timestamp without time zone,
    trip_end_timestamp timestamp without time zone,
    pickup_community_area double precision,
    pickup_centroid_location text,
    dropoff_community_area double precision,
    dropoff_centroid_location text,
    pickup_zip_code double precision,
    dropoff_zip_code double precision,
    pickup_neighborhood text,
    dropoff_neighborhood text
);


ALTER TABLE public.taxi_trips OWNER TO postgres;

--
-- TOC entry 3591 (class 0 OID 0)
-- Dependencies: 4
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: postgres
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO PUBLIC;


-- Completed on 2025-06-07 21:50:24 IST

--
-- PostgreSQL database dump complete
--

