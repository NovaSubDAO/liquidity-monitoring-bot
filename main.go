package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/NovaSubDAO/nova-sdk/go/pkg/sdk"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	OptRPCEndpoint = os.Getenv("OPT_RPC_ENDPOINT")
	ChannelID      = os.Getenv("CHANNEL_ID")
	BotToken       = os.Getenv("BOT_TOKEN")

	PoolAddress = common.HexToAddress("0x131525f3FA23d65DC2B1EB8B6483a28c43B06916")
	UsdcAddress = common.HexToAddress("0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85")
	SdaiAddress = common.HexToAddress("0x2218a117083f5B482B0bB821d27056Ba9c04b1D3")

	UsdcDecimals = 6
	SdaiDecimals = 18

	OptChainId = int64(10)
)

var client *ethclient.Client

func main() {
	var err error
	client, err = ethclient.Dial(OptRPCEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	dg, err := discordgo.New("Bot " + BotToken)
	if err != nil {
		log.Fatalf("error creating Discord session: %v", err)
	}

	dg.AddHandler(onReady)

	err = dg.Open()
	if err != nil {
		log.Fatalf("error opening connection: %v", err)
	}
	defer dg.Close()

	select {}
}

func onReady(s *discordgo.Session, event *discordgo.Ready) {
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sendDataToChannel(s)
}

func getPoolBalance(tokenAddress common.Address, decimals int) float64 {
	funcSig := "balanceOf(address)"
	methodID := crypto.Keccak256([]byte(funcSig))[:4]
	paddedAddress := common.LeftPadBytes(PoolAddress.Bytes(), 32)
	data := append(methodID, paddedAddress...)

	msg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: data,
	}
	resultByte, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		log.Fatal("eth call failed: ", err)
	}

	result := hex.EncodeToString(resultByte)
	if result == "0x" {
		return 0
	}

	balance, success := new(big.Int).SetString(strings.TrimPrefix(result, "0x"), 16)
	if !success {
		log.Fatal("Failed to parse balance")
	}

	divisor := big.NewFloat(0).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	fBalance := new(big.Float).SetInt(balance)
	value := new(big.Float).Quo(fBalance, divisor)
	bal, _ := value.Float64()
	return bal
}

func getPriceAndSlippage() (float64, float64, error) {
	novaSdk, err := sdk.NewNovaSDK(OptRPCEndpoint, OptChainId)
	amount := big.NewInt(1e8)
	slippage, expectedPrice, _, err := novaSdk.GetSlippage("USDC", amount)
	if err != nil {
		return 0, 0, err
	}
	return slippage, expectedPrice, nil
}

func sendDataToChannel(s *discordgo.Session) {
	sdaiBalance := getPoolBalance(SdaiAddress, SdaiDecimals)
	usdcBalance := getPoolBalance(UsdcAddress, UsdcDecimals)

	slippage, expectedPrice, err := getPriceAndSlippage()
	if err != nil {
		log.Printf("Failed to get slippage message: %v", err)
	}

	// Format numbers
	formattedSdaiBalance := formatWithCommas(int64(sdaiBalance))
	formattedUsdcBalance := formatWithCommas(int64(usdcBalance))
	formattedSlippage := fmt.Sprintf("%.4f%%", slippage)
	formattedExpectedPrice := fmt.Sprintf("%.4f", expectedPrice)

	message := fmt.Sprintf("**Velodrome CL1-USDC/sDAI pool**\n- sDAI balance: %s\n- USDC balance: %s\n- Slippage (for 100 USDC): %s\n- Venue Price (USDC/sDAI): %s",
		formattedSdaiBalance, formattedUsdcBalance, formattedSlippage, formattedExpectedPrice)

	_, err = s.ChannelMessageSend(ChannelID, message)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	} else {
		fmt.Println("Message sent successfully")
	}
}

// Helper function to format numbers with commas
func formatWithCommas(value int64) string {
	return humanize.Comma(value)
}
