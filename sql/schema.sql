CREATE TABLE message_info
(
  id         VARCHAR(32) PRIMARY KEY,
  remote_jid VARCHAR(50)                                          NOT NULL,
  from_me    BOOL DEFAULT TRUE                                    NOT NULL,
  type       ENUM ('TEXT', 'AUDIO', 'IMAGE', 'VIDEO', 'DOCUMENT') NOT NULL,
  timestamp  TIMESTAMP
);
CREATE UNIQUE INDEX message_info_id_uindex
  ON message_info (id);

CREATE TABLE media
(
  id              int PRIMARY KEY AUTO_INCREMENT,
  url             TEXT,
  mediakey        TEXT,
  file_enc_sha256 TEXT,
  file_sha256     TEXT,
  data            LONGBLOB NOT NULL
);

CREATE TABLE image
(
  message_id VARCHAR(32) PRIMARY KEY,
  caption    TEXT,
  thumbnail  BLOB,
  mimetype   TEXT,
  media_id   int NOT NULL,
  CONSTRAINT image_message_info_id_fk FOREIGN KEY (message_id) REFERENCES message_info (id),
  CONSTRAINT image_media_id_fk FOREIGN KEY (media_id) REFERENCES media (id)
);