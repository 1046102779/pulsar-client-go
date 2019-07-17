//
// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//

package pulsar

import (
    `context`
    `fmt`
    `github.com/stretchr/testify/assert`
    `io/ioutil`
    `log`
    `net/http`
    `testing`
    `time`
)

func TestProducerConsumer(t *testing.T) {
    client, err := NewClient(ClientOptions{
        URL: "pulsar://localhost:6650",
    })

    assert.Nil(t, err)
    defer client.Close()

    topic := "my-topic"
    ctx := context.Background()

    // create consumer
    consumer, err := client.Subscribe(ConsumerOptions{
        Topic:            topic,
        SubscriptionName: "my-sub",
        Type:             Shared,
    })
    assert.Nil(t, err)
    defer consumer.Close()

    // create producer
    producer, err := client.CreateProducer(ProducerOptions{
        Topic:           topic,
        DisableBatching: false,
    })
    assert.Nil(t, err)
    defer producer.Close()

    // send 10 messages
    for i := 0; i < 10; i++ {
        if err := producer.Send(ctx, &ProducerMessage{
            Payload: []byte(fmt.Sprintf("hello-%d", i)),
        }); err != nil {
            log.Fatal(err)
        }
    }

    // receive 10 messages
    for i := 0; i < 10; i++ {
        msg, err := consumer.Receive(context.Background())
        if err != nil {
            log.Fatal(err)
        }

        expectMsg := fmt.Sprintf("hello-%d", i)
        assert.Equal(t, []byte(expectMsg), msg.Payload())

        // ack message
        if err := consumer.Ack(msg); err != nil {
            log.Fatal(err)
        }
    }

    // unsubscribe consumer
    if err := consumer.Unsubscribe(); err != nil {
        log.Fatal(err)
    }
}

func TestConsumerConnectError(t *testing.T) {
    client, err := NewClient(ClientOptions{
        URL: "pulsar://invalid-hostname:6650",
    })

    assert.Nil(t, err)

    defer client.Close()

    consumer, err := client.Subscribe(ConsumerOptions{
        Topic:            "my-topic",
        SubscriptionName: "my-subscription",
    })

    // Expect error in creating consumer
    assert.Nil(t, consumer)
    assert.NotNil(t, err)

    assert.Equal(t, err.Error(), "connection error")
}

func TestConsumerWithInvalidConf(t *testing.T) {
    client, err := NewClient(ClientOptions{
        URL: "pulsar://localhost:6650",
    })

    if err != nil {
        t.Fatal(err)
        return
    }

    defer client.Close()

    consumer, err := client.Subscribe(ConsumerOptions{
        Topic: "my-topic",
    })

    // Expect error in creating cosnumer
    assert.Nil(t, consumer)
    assert.NotNil(t, err)

    fmt.Println(err.Error())
    assert.Equal(t, err.(*Error).Result(), SubscriptionNotFound)

    consumer, err = client.Subscribe(ConsumerOptions{
        SubscriptionName: "my-subscription",
    })

    // Expect error in creating consumer
    assert.Nil(t, consumer)
    assert.NotNil(t, err)

    assert.Equal(t, err.(*Error).Result(), TopicNotFound)
}

func TestConsumer_SubscriptionInitPos(t *testing.T) {
    client, err := NewClient(ClientOptions{
        URL: "pulsar://localhost:6650",
    })

    assert.Nil(t, err)
    defer client.Close()

    topicName := fmt.Sprintf("testSeek-%d", time.Now().Unix())
    subName := "test-subscription-initial-earliest-position"

    // create producer
    producer, err := client.CreateProducer(ProducerOptions{
        Topic: topicName,
    })
    assert.Nil(t, err)
    defer producer.Close()

    //sent message
    ctx := context.Background()

    err = producer.Send(ctx, &ProducerMessage{
        Payload: []byte("msg-1-content-1"),
    })
    assert.Nil(t, err)

    err = producer.Send(ctx, &ProducerMessage{
        Payload: []byte("msg-1-content-2"),
    })
    assert.Nil(t, err)

    // create consumer
    consumer, err := client.Subscribe(ConsumerOptions{
        Topic:               topicName,
        SubscriptionName:    subName,
        SubscriptionInitPos: Earliest,
    })
    assert.Nil(t, err)
    defer consumer.Close()

    msg, err := consumer.Receive(ctx)
    assert.Nil(t, err)

    assert.Equal(t, "msg-1-content-1", string(msg.Payload()))
}

func makeHttpPutCall(t *testing.T, url string) string {
    return makeHttpCall(t, http.MethodPut, url)
}

func makeHttpGetCall(t *testing.T, url string) string {
    return makeHttpCall(t, http.MethodGet, url)
}

func makeHttpCall(t *testing.T, method string, url string) string {
    client := http.Client{}

    req, err := http.NewRequest(method, url, nil)
    if err != nil {
        t.Fatal(err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    res, err := client.Do(req)
    if err != nil {
        t.Fatal(err)
    }

    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        t.Fatal(err)
    }

    return string(body)
}

func TestConsumerShared(t *testing.T) {
    client, err := NewClient(ClientOptions{
        URL: "pulsar://localhost:6650",
    })
    assert.Nil(t, err)
    defer client.Close()

    topic := "persistent://public/default/test-topic-6"

    consumer1, err := client.Subscribe(ConsumerOptions{
        Topic:            topic,
        SubscriptionName: "sub-1",
        Type:             KeyShared,
    })
    assert.Nil(t, err)
    defer consumer1.Close()

    consumer2, err := client.Subscribe(ConsumerOptions{
        Topic:            topic,
        SubscriptionName: "sub-1",
        Type:             KeyShared,
    })
    assert.Nil(t, err)
    defer consumer2.Close()

    // create producer
    producer, err := client.CreateProducer(ProducerOptions{
        Topic:    topic,
        //Batching: false,
    })
    assert.Nil(t, err)
    defer producer.Close()

    ctx := context.Background()
    for i := 0; i < 10; i++ {
        err := producer.Send(ctx, &ProducerMessage{
            Key:     fmt.Sprintf("key-shared-%d", i%3),
            Payload: []byte(fmt.Sprintf("value-%d", i)),
        })
        assert.Nil(t, err)
    }

    time.Sleep(time.Second * 5)

    go func() {
        for i := 0; i < 10; i++ {
            msg, err := consumer1.Receive(ctx)
            assert.Nil(t, err)
            if msg != nil {
                fmt.Printf("consumer1 key is: %s, value is: %s\n", msg.Key(), string(msg.Payload()))
                err = consumer1.Ack(msg)
                assert.Nil(t, err)
            }
        }
    }()

    go func() {
        for i := 0; i < 10; i++ {
            msg2, err := consumer2.Receive(ctx)
            assert.Nil(t, err)
            if msg2 != nil {
                fmt.Printf("consumer2 key is:%s, value is: %s\n", msg2.Key(), string(msg2.Payload()))
                err = consumer2.Ack(msg2)
                assert.Nil(t, err)
            }
        }
    }()
}
