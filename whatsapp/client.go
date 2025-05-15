package whatsapp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/mattn/go-sqlite3"
)

// Client representa o cliente WhatsApp
type Client struct {
	WAClient     *whatsmeow.Client
	eventHandler func(interface{})
}

// NewClient cria uma nova instância do cliente WhatsApp
func NewClient(sessionPath string) (*Client, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)

	// Abrir container para armazenar sessão
	container, err := sqlstore.New("sqlite3", fmt.Sprintf("file:%s/store.db?_foreign_keys=on", sessionPath), dbLog)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar container: %v", err)
	}

	// Obter dispositivo
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter device: %v", err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	return &Client{
		WAClient: client,
	}, nil
}

// Connect conecta ao WhatsApp e gerencia o QR code se necessário
func (c *Client) Connect() error {
	if c.WAClient.Store.ID == nil {
		// Nova sessão precisa de QR code
		qrChan, _ := c.WAClient.GetQRChannel(context.Background())
		err := c.WAClient.Connect()
		if err != nil {
			return err
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("Escaneie o QR code a seguir:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("Você pode pressionar Ctrl+C para cancelar.")
			} else {
				fmt.Println("Login state:", evt.Event)
			}
		}
	} else {
		// Sessão já existe
		err := c.WAClient.Connect()
		if err != nil {
			return err
		}
		fmt.Println("Conectado com sucesso usando sessão existente!")
	}

	return nil
}

// SetEventHandler configura o handler de eventos
func (c *Client) SetEventHandler(handler func(interface{})) {
	c.eventHandler = handler
	c.WAClient.AddEventHandler(handler)
}

// SendTextMessage envia uma mensagem de texto para um destinatário
func (c *Client) SendTextMessage(recipient string, message string) error {
	// Validar número de telefone
	var jid types.JID
	if strings.Contains(recipient, "@") {
		// Se já estiver no formato JID
		parsedJID, err := types.ParseJID(recipient)
		if err != nil {
			return fmt.Errorf("número de telefone inválido: %s", recipient)
		}
		jid = parsedJID
	} else {
		// Se for apenas o número
		jid = types.NewJID(recipient, types.DefaultUserServer)
	}

	// Criar mensagem usando proto
	msg := &waProto.Message{
		Conversation: proto.String(message),
	}

	// Enviar mensagem
	_, err := c.WAClient.SendMessage(context.Background(), jid, msg)
	return err
}

// Close fecha a conexão e salva a sessão
func (c *Client) Close() {
	c.WAClient.Disconnect()
}

// AddCustomEventHandler adiciona um manipulador de eventos personalizado ao cliente
// Este método é necessário para o recurso de monitoramento de usuários em grupos
func (c *Client) AddCustomEventHandler(handler func(interface{})) {
	// Adiciona o handler personalizado ao cliente WhatsApp
	c.WAClient.AddEventHandler(handler)
}
