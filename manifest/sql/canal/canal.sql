-- Active: 1761738628982@@localhost@3306@mysql
-- Active: 1761738628982@@localhost@3306@Natsume
CREATE USER 'canal'@'%' IDENTIFIED BY 'canal';

GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'canal'@'%';
GRANT SELECT, SHOW VIEW ON *.* TO 'canal'@'%';
GRANT SELECT, SHOW VIEW ON performance_schema.* TO 'canal'@'%';
GRANT SELECT, SHOW VIEW ON information_schema.* TO 'canal'@'%';
GRANT SELECT, SHOW VIEW ON Natsume.* TO 'canal'@'%';
GRANT SELECT              ON mysql.*            TO 'canal'@'%';
FLUSH PRIVILEGES;

