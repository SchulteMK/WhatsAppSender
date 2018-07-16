create procedure insert_image(IN pId        varchar(32), IN pRemoteJid varchar(50), IN pFromMe tinyint(1),
                              IN pTimestamp timestamp, IN pCaption text, IN pThumbnail blob, IN pMimetype text,
                              IN pUrl       text, IN pMediakey text, IN pFile_enc_sha256 text, IN pFile_sha256 text,
                              IN pData      longblob)
  BEGIN
    IF (SELECT COUNT(*)
        FROM message_info
        WHERE id = pId) = 0
    THEN
      INSERT INTO message_info (id, remote_jid, from_me, timestamp, type)
      VALUES (pId, pRemoteJid, pFromMe, pTimestamp, 'IMAGE');

      INSERT INTO media (url, mediakey, file_enc_sha256, file_sha256, data)
      VALUES (pUrl, pMediakey, pFile_enc_sha256, pFile_sha256, pData);

      INSERT INTO image (message_id, caption, thumbnail, mimetype, media_id)
      VALUES (pId, pCaption, pThumbnail, pMimetype, LAST_INSERT_ID());
    END IF;
  END;

