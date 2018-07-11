create procedure insert_text(IN pId        varchar(32), IN pRemoteJid varchar(50), IN pFromMe tinyint(1),
                             IN pTimestamp timestamp, IN pText text)
  BEGIN
    IF (SELECT COUNT(*)
        FROM message_info
        WHERE id = pId) = 0
    THEN
      INSERT INTO message_info (id, remote_jid, from_me, timestamp)
      VALUES (pId, pRemoteJid, pFromMe, pTimestamp);
      INSERT INTO message_type (message_id, type) VALUES (pId, 'Text');
      INSERT INTO text (message_id, text) VALUES (pId, pText);
    END IF;
  END;

