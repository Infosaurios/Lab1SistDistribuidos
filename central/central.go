package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	pb "Tarea1-Grupo24/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// labRenca, labPohang, labPripiat, labKampala
var (
	puertos  = [4]string{":50051", ":50055", ":50059", ":50063"}
	equipo1_ = true //disponibilidad
	equipo2_ = true
	cont=1
	cont2=1
	helpQueue = "SOS"       //Nombre de la cola
	hostQ     = "localhost" //Host de RabbitMQ
	hostS     = [4]string{"dist093","dist094","dist095","dist096"} //Host de un Laboratorio
)

func failOnError(err error, msg string) {
	if err != nil {
		fmt.Println(err)
	}
}

func main() {

	conn, err := amqp.Dial("amqp://guest:guest@" + hostQ + ":5672") //Conexion con RabbitMQ
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()
	f, errf := os.Create("SOLICITUDES.txt")
	check(errf)
    defer f.Close()
	q, err := ch.QueueDeclare(
		helpQueue, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to declare a queue")

	chDelivery, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	var forever chan struct{}
	port := "" //puerto de la conexion con el laboratorio
	host:=""

	for delivery := range chDelivery {

		labName := string(delivery.Body)
		fmt.Println("Mensaje asíncrono de laboratorio " + labName + " leído") //obtiene el primer mensaje de la cola

		if labName == "Renca (la lleva) - Chile" {
			port = puertos[0]
			host = hostS[0]
		} else if labName == "Pohang - Korea" {
			port = puertos[1]
			host = hostS[1]
		} else if labName == "Pripiat - Rusia" {
			port = puertos[2]
			host = hostS[2]
		} else if labName == "Kampala - Uganda" {
			port = puertos[3]
			host = hostS[3]
		}
		/** Crea la conexion sincrona con el laboratorio **/
		connS, err := grpc.Dial(host+port, grpc.WithInsecure())

		if err != nil {
			panic("No se pudo conectar con el servidor" + err.Error())
		}

		serviceCliente := pb.NewMessageServiceClient(connS)

		/*****************Estado Contencion*********************/

		//Manda info a los lab sobre la disp de los equipos de mercenarios
		resDisp, errDisp := serviceCliente.CheckDispEscuadron(
			context.Background(),
			&pb.Escuadron{
				Equipo1: equipo1_, Equipo2: equipo2_,
			})
		if errDisp != nil {
			panic("No se puede crear el mensaje " + err.Error())
		}

		//Recibe el nombre del equipo elegido y el nombre del lab. El equipo elegido
		//cambia su valor a false => equipo está ocupado

		go func() {
			if resDisp.Equipox == "Escuadra 1" {
				equipo1_ = false
				primeraLlegada := true
				escuadronNoListo := true
				fmt.Println("Se envía " + resDisp.Equipox + " a Laboratorio " + resDisp.NombreLab)
				for escuadronNoListo {
					//Se manda el escuadron al lab
					res, err := serviceCliente.ContencionStatus(
						context.Background(),
						&pb.EquipoEnviadoPorCentral{
							Eepc:           resDisp.Equipox, //se manda el equipo al laboratorio
							PrimeraLlegada: primeraLlegada,
						})
					if err != nil {
						panic("No se puede crear el mensaje " + err.Error())
					}
					//Se recibe el estado de contencion y nombre del escuadron
					//Si el estallido está contenido, se cierra la conexión con el lab
					if res.Status.String() == "NOLISTO" {
						fmt.Println("Status " + res.NombreEscuadron + ": " + "["+res.Status.String()+"]")
						cont=cont+1
						time.Sleep(5 * time.Second) //espera de 5 segundos
						primeraLlegada = false
					} else {
						escuadronNoListo = false
						fmt.Println("Status " + res.NombreEscuadron + ": " + "["+res.Status.String()+"]")
						fmt.Println("Retorno a Central " + res.NombreEscuadron + ", Conexión Laboratorio " + resDisp.NombreLab + " Cerrada")
						equipo1_ = true //vuelve a quedar disponible
						f.WriteString(resDisp.NombreLab+";"+strconv.Itoa(cont)+"\n")
						cont=1
						connS.Close()   //Se cierra la conexión
					}
				}
			}
		}()

		go func() {
			if resDisp.Equipox == "Escuadra 2" {
				equipo2_ = false
				primeraLlegada := true
				escuadronNoListo := true
				fmt.Println("Se envía " + resDisp.Equipox + " a Laboratorio " + resDisp.NombreLab)
				for escuadronNoListo {
					//Se manda el escuadron al lab
					res, err := serviceCliente.ContencionStatus(
						context.Background(),
						&pb.EquipoEnviadoPorCentral{
							Eepc:           resDisp.Equipox, //se manda el equipo al laboratorio
							PrimeraLlegada: primeraLlegada,
						})
					if err != nil {
						panic("No se puede crear el mensaje " + err.Error())
					}
					//Se recibe el estado de contencion y nombre del escuadron
					//Si el estallido está contenido, se cierra la conexión con el lab
					if res.Status.String() == "NOLISTO" {
						fmt.Println("Status " + res.NombreEscuadron + ": " + "["+res.Status.String()+"]")
						cont2=cont2+1
						time.Sleep(5 * time.Second) //espera de 5 segundos
						primeraLlegada = false
					} else {
						escuadronNoListo = false
						fmt.Println("Status " + res.NombreEscuadron + ": " + "["+res.Status.String()+"]")
						fmt.Println("Retorno a Central " + res.NombreEscuadron + ", Conexión Laboratorio " + resDisp.NombreLab + " Cerrada")
						equipo2_ = true //vuelve a quedar disponible
						f.WriteString(resDisp.NombreLab+";"+strconv.Itoa(cont2)+"\n")
						cont2=1
						connS.Close()   //Se cierra la conexión
					}
				}
			}
		}()

		
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for sig := range c {
				// sig is a ^C, handle it
				// Proto mande msg manera sincrona a todos los labs
				
				r,_:=serviceCliente.FinPrograma(
					context.Background(),
					&pb.MessageTermino{
						EndSignal: true,
						MsgFin:    "Lab termine su ejecucion",
					})
				time.Sleep(1*time.Second)
				
				fmt.Println("SEACABO!!")
				os.Exit(1)
				fmt.Println(r)
				fmt.Println(sig)
			}
		}()
	}

	fmt.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
	
func check(e error) {
    if e != nil {
        panic(e)
    }
}