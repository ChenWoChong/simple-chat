CREATE DATABASE `test` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
grant all PRIVILEGES on test.* to test@'%' identified by '123456';
flush privileges;
use test;