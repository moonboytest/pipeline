// Напишите код, реализующий пайплайн, работающий с целыми числами и состоящий из следующих стадий:

// Стадия фильтрации отрицательных чисел (не пропускать отрицательные числа).

// Стадия фильтрации чисел, не кратных 3 (не пропускать такие числа), исключая также и 0.

// Стадия буферизации данных в кольцевом буфере с интерфейсом, соответствующим тому,
// который был дан в качестве задания в 19 модуле.
// В этой стадии предусмотреть опустошение буфера
// (и соответственно, передачу этих данных, если они есть, дальше) с определённым интервалом во времени.
// Значения размера буфера и этого интервала времени сделать настраиваемыми
// (как мы делали: через константы или глобальные переменные).

// Написать источник данных для конвейера. Непосредственным источником данных должна быть консоль.

// Также написать код потребителя данных конвейера.
// Данные от конвейера можно направить снова в консоль построчно,
// сопроводив их каким-нибудь поясняющим текстом, например: «Получены данные …».

// При написании источника данных подумайте о фильтрации нечисловых данных,
// которые можно ввести через консоль. Как и где их фильтровать, решайте сами.

// Готовый код присылайте на проверку ментором в поле ниже.
// Если совсем не понимаете, с чего начать и как решать задачу, загляните в следующий юнит,
// а потом возвращайтесь со своим решением, чтобы отправить его ментору.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Интервал очистки буфера
const timeInterval time.Duration = 5 * time.Second

// Размер буфера
const bufferSize int = 5

// Структура буфера
type RingBuffer struct {
	array    []int
	position int
	size     int
	m        sync.Mutex
}

//----------------------------------------
// 			  Методы буфера			     |
//----------------------------------------

// Инициализация буфера
func NewBuffer(size int) *RingBuffer {

	return &RingBuffer{make([]int, size), -1, size, sync.Mutex{}}
}

// Добавление элемента в буфер
func (rb *RingBuffer) Add(num int) {
	rb.m.Lock()
	defer rb.m.Unlock()

	if rb.position == rb.size-1 {
		for i := 1; i <= rb.size; i++ {
			rb.array[i-1] = rb.array[i]
		}
		rb.array[rb.position] = num
	} else {
		rb.position++
		rb.array[rb.position] = num
	}

}

// Метод получения элементов из буфера
func (rb *RingBuffer) Get() []int {

	if rb.position == 0 {
		return nil
	}

	rb.m.Lock()
	defer rb.m.Unlock()

	var output []int = rb.array[:rb.position+1]

	rb.position = -1

	return output

}

//-------------------------------------------
// 	Стадия конвейера, обрабабатывающая числа |
//               и ее методы				 |
//-------------------------------------------

// Стадия конвейера, обрабатывающая числа
type stageInt func(<-chan bool, <-chan int) <-chan int

// Функция старта стадии конвейера
func (p *Pipeline) RunStage(stage stageInt, sourceChan <-chan int) <-chan int {

	return stage(p.done, sourceChan)
}

//-------------------------------------------
// 	               Пайплайн                  |
//               и его методы				 |
//-------------------------------------------

// Структура пайплайна
type Pipeline struct {
	stages []stageInt
	done   <-chan bool
}

// Новый экземпляр пайплайна
func NewPipeline(done <-chan bool, stages ...stageInt) *Pipeline {
	return &Pipeline{stages: stages, done: done}
}

// Запуск пайплайна
func (p *Pipeline) Start(source <-chan int) <-chan int {
	var c <-chan int = source

	for index := range p.stages {
		c = p.RunStage(p.stages[index], c)
	}
	return c
}

//-------------------------------------------
// 	               Ввод чисел                |
//-------------------------------------------

func Input() (<-chan int, <-chan bool) {
	c := make(chan int)
	done := make(chan bool)

	go func() {
		defer close(done)

		scanner := bufio.NewScanner(os.Stdin)
		var data string

		for {
			scanner.Scan()
			data = scanner.Text()
			if strings.EqualFold(data, "exit") {
				fmt.Println("Выходим из программы")
				return
			}
			i, err := strconv.Atoi(data)
			if err != nil {
				fmt.Println("Программа обрабатывает только целые числа")
				continue
			}
			c <- i
		}
	}()
	return c, done
}

//-------------------------------------------
// 	               Фильтрация                |
//-------------------------------------------

// Фильтрация отрицательных чисел
func negativeFilter(done <-chan bool, c <-chan int) <-chan int {
	convertedIntChan := make(chan int)
	go func() {
		for {
			select {
			case data := <-c:
				if data > 0 {
					select {
					case convertedIntChan <- data:
					case <-done:
						return
					}
				}
			case <-done:
				return
			}
		}
	}()
	return convertedIntChan
}

// Фильтрация чисел не кратных трем
func specialFilter(done <-chan bool, c <-chan int) <-chan int {
	convertedIntChan := make(chan int)
	go func() {
		for {
			select {
			case data := <-c:
				if data%3 == 0 && data != 0 {
					select {
					case convertedIntChan <- data:
					case <-done:
						return
					}
				}
			case <-done:
				return
			}
		}
	}()
	return convertedIntChan
}

//-------------------------------------------
// 	               Буферизация               |
//-------------------------------------------

func bufferStageInt(done <-chan bool, c <-chan int) <-chan int {
	bufferIntChan := make(chan int)
	buffer := NewBuffer(bufferSize)

	go func() {
		for {
			select {
			case data := <-c:
				buffer.Add(data)
			case <-done:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case <-time.After(timeInterval):
				bufferData := buffer.Get()
				if bufferData != nil {
					for _, data := range bufferData {
						select {
						case bufferIntChan <- data:
						case <-done:
							return
						}
					}
				}
			case <-done:
				return
			}
		}
	}()
	return bufferIntChan
}

//-------------------------------------------
// 	            Потребитель данных          |
//-------------------------------------------

func consumer(done <-chan bool, c <-chan int) {
	for {
		select {
		case data := <-c:
			fmt.Printf("Обработаны данные: %d\n", data)
		case <-done:
			return
		}
	}
}

func main() {

	source, done := Input()

	Pipeline := NewPipeline(done, negativeFilter, specialFilter, bufferStageInt)

	consumer(done, Pipeline.Start(source))
}
