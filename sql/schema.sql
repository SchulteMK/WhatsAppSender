create table media
(
  id        int auto_increment
    primary key,
  caption   mediumtext null,
  mimetype  blob       null,
  thumbnail blob       null,
  data      longblob   not null
);

create table message_info
(
  id         varchar(32)                         not null,
  remote_jid varchar(50)                         not null,
  from_me    tinyint(1) default '1'              not null,
  timestamp  timestamp default CURRENT_TIMESTAMP not null
  on update CURRENT_TIMESTAMP,
  media      tinyint(1) default '0'              not null,
  media_id   int                                 null,
  constraint message_info_id_uindex
  unique (id),
  constraint message_info_media_id_fk
  foreign key (media_id) references media (id)
);

alter table message_info
  add primary key (id);

create table text
(
  message_id varchar(32) not null primary key,
  text       mediumtext  null,
  constraint text_message_info_id_fk
  foreign key (message_id) references message_info (id)
);

create procedure insert_media(IN pId        varchar(32), IN pRemoteJid varchar(50), IN pFromMe tinyint(1),
                              IN pTimestamp timestamp, IN pCaption text, IN pThumbnail blob, IN pMimetype text,
                              IN pData      longblob)
  BEGIN
    IF (SELECT COUNT(*) FROM message_info WHERE id = pId) = 0
    THEN
      INSERT INTO media(caption, thumbnail, mimetype, data) VALUES (pCaption, pThumbnail, pMimetype, pData);

      INSERT INTO message_info(id, remote_jid, from_me, timestamp, media, media_id)
      VALUES (pId, pRemoteJid, pFromMe, pTimestamp, TRUE, LAST_INSERT_ID());

    END IF;
  END;

create procedure insert_text(IN pId        varchar(32), IN pRemoteJid varchar(50), IN pFromMe tinyint(1),
                             IN pTimestamp timestamp, IN pText text)
  BEGIN
    IF (SELECT COUNT(*) FROM message_info WHERE id = pId) = 0
    THEN
      INSERT INTO message_info(id, remote_jid, from_me, timestamp) VALUES (pId, pRemoteJid, pFromMe, pTimestamp);
      INSERT INTO text(message_id, text) VALUES (pId, pText);
    END IF;
  END;
