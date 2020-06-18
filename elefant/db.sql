--
-- PostgreSQL database dump
--

-- Dumped from database version 11.6
-- Dumped by pg_dump version 12.3

-- Started on 2020-06-19 01:40:06

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 2 (class 3079 OID 16425)
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- TOC entry 3889 (class 0 OID 0)
-- Dependencies: 2
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


SET default_tablespace = '';

--
-- TOC entry 200 (class 1259 OID 16561)
-- Name: account; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account (
    id uuid NOT NULL,
    client uuid NOT NULL,
    "time" timestamp without time zone NOT NULL,
    balance double precision NOT NULL,
    currency character(3) NOT NULL,
    revision bigint NOT NULL
);


--
-- TOC entry 198 (class 1259 OID 16522)
-- Name: auth_token; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.auth_token (
    id integer NOT NULL,
    token uuid NOT NULL,
    client uuid NOT NULL,
    "time" timestamp without time zone NOT NULL,
    update timestamp without time zone NOT NULL,
    token_prev uuid,
    request json
);


--
-- TOC entry 197 (class 1259 OID 16520)
-- Name: auth-token_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public."auth-token_id_seq"
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 3890 (class 0 OID 0)
-- Dependencies: 197
-- Name: auth-token_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public."auth-token_id_seq" OWNED BY public.auth_token.id;


--
-- TOC entry 199 (class 1259 OID 16540)
-- Name: client; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.client (
    id uuid NOT NULL,
    email text NOT NULL,
    password text NOT NULL,
    "time" timestamp without time zone NOT NULL,
    request json,
    confirmed boolean NOT NULL,
    name text NOT NULL
);


--
-- TOC entry 201 (class 1259 OID 17514)
-- Name: client_confirmation; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.client_confirmation (
    id uuid NOT NULL,
    "time" timestamp without time zone NOT NULL,
    token text NOT NULL,
    client uuid NOT NULL
);


--
-- TOC entry 3733 (class 2604 OID 16525)
-- Name: auth_token id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token ALTER COLUMN id SET DEFAULT nextval('public."auth-token_id_seq"'::regclass);


--
-- TOC entry 3752 (class 2606 OID 16567)
-- Name: account acc-currency_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account
    ADD CONSTRAINT "acc-currency_unq" UNIQUE (client, currency);


--
-- TOC entry 3754 (class 2606 OID 16565)
-- Name: account account_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account
    ADD CONSTRAINT account_pkey PRIMARY KEY (id);


--
-- TOC entry 3735 (class 2606 OID 16556)
-- Name: auth_token auth-token-client_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token-client_unq" UNIQUE (client, token);


--
-- TOC entry 3737 (class 2606 OID 16560)
-- Name: auth_token auth-token-prev-client_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token-prev-client_unq" UNIQUE (client, token_prev);


--
-- TOC entry 3739 (class 2606 OID 16558)
-- Name: auth_token auth-token-prev_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token-prev_unq" UNIQUE (token_prev);


--
-- TOC entry 3741 (class 2606 OID 16527)
-- Name: auth_token auth-token_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token_pkey" PRIMARY KEY (id);


--
-- TOC entry 3743 (class 2606 OID 16529)
-- Name: auth_token auth-token_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token_unq" UNIQUE (token);


--
-- TOC entry 3747 (class 2606 OID 16549)
-- Name: client client-email_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client
    ADD CONSTRAINT "client-email_unq" UNIQUE (email);


--
-- TOC entry 3749 (class 2606 OID 16547)
-- Name: client client_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client
    ADD CONSTRAINT client_pkey PRIMARY KEY (id);


--
-- TOC entry 3757 (class 2606 OID 17529)
-- Name: client_confirmation confirmation-token_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_confirmation
    ADD CONSTRAINT "confirmation-token_unq" UNIQUE (client, token);


--
-- TOC entry 3759 (class 2606 OID 17521)
-- Name: client_confirmation confirmation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_confirmation
    ADD CONSTRAINT confirmation_pkey PRIMARY KEY (id);


--
-- TOC entry 3750 (class 1259 OID 16574)
-- Name: acc-client-rev_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "acc-client-rev_idx" ON public.account USING btree (client, revision);


--
-- TOC entry 3744 (class 1259 OID 16630)
-- Name: client-confirmed-id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "client-confirmed-id_idx" ON public.client USING btree (confirmed, id);


--
-- TOC entry 3745 (class 1259 OID 16629)
-- Name: client-email-confirmed_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "client-email-confirmed_idx" ON public.client USING btree (confirmed, email);


--
-- TOC entry 3755 (class 1259 OID 17527)
-- Name: confirmation-time_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "confirmation-time_idx" ON public.client_confirmation USING btree ("time");


--
-- TOC entry 3761 (class 2606 OID 16568)
-- Name: account acc-client_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account
    ADD CONSTRAINT "acc-client_ref" FOREIGN KEY (client) REFERENCES public.client(id) ON DELETE CASCADE;


--
-- TOC entry 3760 (class 2606 OID 16550)
-- Name: auth_token auth-token-client_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token-client_ref" FOREIGN KEY (client) REFERENCES public.client(id) ON DELETE CASCADE NOT VALID;


--
-- TOC entry 3762 (class 2606 OID 17522)
-- Name: client_confirmation confirmation-client_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_confirmation
    ADD CONSTRAINT "confirmation-client_ref" FOREIGN KEY (client) REFERENCES public.client(id) ON DELETE CASCADE;


-- Completed on 2020-06-19 01:40:21

--
-- PostgreSQL database dump complete
--

