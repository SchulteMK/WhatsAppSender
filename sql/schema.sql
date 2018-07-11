CREATE TABLE message_info
(
  id         VARCHAR(32) PRIMARY KEY,
  remote_jid VARCHAR(50)       NOT NULL,
  from_me    BOOL DEFAULT TRUE NOT NULL,
  timestamp  TIMESTAMP
);
CREATE UNIQUE INDEX message_info_id_uindex
  ON message_info (id);


CREATE TABLE message_type
(
  message_id varchar(32) PRIMARY KEY                              NOT NULL,
  type       ENUM ('TEXT', 'AUDIO', 'IMAGE', 'VIDEO', 'DOCUMENT') NOT NULL,
  CONSTRAINT message_type_message_info_id_fk FOREIGN KEY (message_id) REFERENCES message_info (id)
);

CREATE TABLE text
(
  message_id varchar(32) PRIMARY KEY NOT NULL,
  text       text,
  CONSTRAINT text_message_info_id_fk FOREIGN KEY (message_id) REFERENCES message_info (id)
);