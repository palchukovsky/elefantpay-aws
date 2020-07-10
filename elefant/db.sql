--
-- PostgreSQL database dump
--

-- Dumped from database version 11.6
-- Dumped by pg_dump version 12.3

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
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


SET default_tablespace = '';

--
-- Name: acc; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.acc (
    id uuid NOT NULL,
    client uuid NOT NULL,
    "time" timestamp without time zone NOT NULL,
    balance double precision NOT NULL,
    currency character(3) NOT NULL,
    revision bigint NOT NULL
);


--
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
-- Name: auth-token_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public."auth-token_id_seq" OWNED BY public.auth_token.id;


--
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
-- Name: client_confirm; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.client_confirm (
    id uuid NOT NULL,
    "time" timestamp without time zone NOT NULL,
    token text NOT NULL,
    client uuid NOT NULL
);


--
-- Name: method; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.method (
    id uuid NOT NULL,
    client uuid,
    info json NOT NULL,
    currency character(3) NOT NULL,
    "time" timestamp without time zone NOT NULL,
    key text NOT NULL,
    type smallint NOT NULL
);


--
-- Name: trans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trans (
    id uuid NOT NULL,
    method uuid NOT NULL,
    acc uuid NOT NULL,
    value double precision NOT NULL,
    "time" timestamp without time zone NOT NULL
);


--
-- Name: auth_token id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token ALTER COLUMN id SET DEFAULT nextval('public."auth-token_id_seq"'::regclass);


--
-- Name: acc acc-currency_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acc
    ADD CONSTRAINT "acc-currency_unq" UNIQUE (client, currency);


--
-- Name: acc account_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acc
    ADD CONSTRAINT account_pkey PRIMARY KEY (id);


--
-- Name: auth_token auth-token-client_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token-client_unq" UNIQUE (client, token);


--
-- Name: auth_token auth-token-prev-client_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token-prev-client_unq" UNIQUE (client, token_prev);


--
-- Name: auth_token auth-token-prev_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token-prev_unq" UNIQUE (token_prev);


--
-- Name: auth_token auth-token_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token_pkey" PRIMARY KEY (id);


--
-- Name: auth_token auth-token_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token_unq" UNIQUE (token);


--
-- Name: client client-email_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client
    ADD CONSTRAINT "client-email_unq" UNIQUE (email);


--
-- Name: client client_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client
    ADD CONSTRAINT client_pkey PRIMARY KEY (id);


--
-- Name: client_confirm confirmation-token_unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_confirm
    ADD CONSTRAINT "confirmation-token_unq" UNIQUE (client, token);


--
-- Name: client_confirm confirmation_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_confirm
    ADD CONSTRAINT confirmation_pkey PRIMARY KEY (id);


--
-- Name: method method-unique-unq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.method
    ADD CONSTRAINT "method-unique-unq" UNIQUE (type, client, key, currency);


--
-- Name: method source_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.method
    ADD CONSTRAINT source_pkey PRIMARY KEY (id);


--
-- Name: trans trans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trans
    ADD CONSTRAINT trans_pkey PRIMARY KEY (id);


--
-- Name: acc-client-rev_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "acc-client-rev_idx" ON public.acc USING btree (client, revision);


--
-- Name: acc-client_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX "acc-client_idx" ON public.acc USING btree (client, id);


--
-- Name: client-confirmed-id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "client-confirmed-id_idx" ON public.client USING btree (confirmed, id);


--
-- Name: client-email-confirmed_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "client-email-confirmed_idx" ON public.client USING btree (confirmed, email);


--
-- Name: confirmation-time_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "confirmation-time_idx" ON public.client_confirm USING btree ("time");


--
-- Name: trans-acc_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "trans-acc_idx" ON public.trans USING btree (acc, "time");


--
-- Name: acc acc-client_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.acc
    ADD CONSTRAINT "acc-client_ref" FOREIGN KEY (client) REFERENCES public.client(id) ON DELETE CASCADE;


--
-- Name: auth_token auth-token-client_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.auth_token
    ADD CONSTRAINT "auth-token-client_ref" FOREIGN KEY (client) REFERENCES public.client(id) ON DELETE CASCADE NOT VALID;


--
-- Name: client_confirm confirmation-client_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_confirm
    ADD CONSTRAINT "confirmation-client_ref" FOREIGN KEY (client) REFERENCES public.client(id) ON DELETE CASCADE;


--
-- Name: method source-client_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.method
    ADD CONSTRAINT "source-client_ref" FOREIGN KEY (client) REFERENCES public.client(id) ON DELETE CASCADE;


--
-- Name: trans trans-acc_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trans
    ADD CONSTRAINT "trans-acc_ref" FOREIGN KEY (acc) REFERENCES public.acc(id) ON DELETE CASCADE;


--
-- Name: trans trans-method_ref; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trans
    ADD CONSTRAINT "trans-method_ref" FOREIGN KEY (method) REFERENCES public.method(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

