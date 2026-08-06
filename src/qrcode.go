package main

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"strings"
)

func qrV4L(data []byte) ([][]bool, error) {
	const size = 33
	const dataCodewords = 80
	const eccCodewords = 20
	if len(data) > 78 { return nil, fmt.Errorf("QR payload too long: %d bytes (max 78)", len(data)) }
	bits := make([]bool,0,dataCodewords*8)
	appendBits := func(value uint,n int){ for i:=n-1;i>=0;i-- { bits=append(bits,((value>>i)&1)!=0) } }
	appendBits(0x4,4); appendBits(uint(len(data)),8)
	for _,b:=range data { appendBits(uint(b),8) }
	remaining:=dataCodewords*8-len(bits); term:=4; if remaining<term { term=remaining }; appendBits(0,term)
	for len(bits)%8!=0 { bits=append(bits,false) }
	code:=make([]byte,0,dataCodewords+eccCodewords)
	for i:=0;i<len(bits);i+=8 { var b byte; for j:=0;j<8;j++ { if bits[i+j] { b|=1<<(7-j) } }; code=append(code,b) }
	pads:=[]byte{0xEC,0x11}; for i:=0;len(code)<dataCodewords;i++ { code=append(code,pads[i&1]) }
	code=append(code,rsRemainder(code,eccCodewords)...)
	modules:=make([][]int8,size); function:=make([][]bool,size)
	for y:=0;y<size;y++ { modules[y]=make([]int8,size); function[y]=make([]bool,size); for x:=0;x<size;x++ { modules[y][x]=-1 } }
	setFunc:=func(x,y int,black bool){ if x<0||y<0||x>=size||y>=size{return}; if black{modules[y][x]=1}else{modules[y][x]=0}; function[y][x]=true }
	drawFinder:=func(cx,cy int){ for dy:=-4;dy<=4;dy++ { for dx:=-4;dx<=4;dx++ { x,y:=cx+dx,cy+dy; if x<0||y<0||x>=size||y>=size{continue}; d:=abs(max(abs(dx),abs(dy))); setFunc(x,y,d!=2&&d!=4) } } }
	drawAlignment:=func(cx,cy int){ for dy:=-2;dy<=2;dy++ { for dx:=-2;dx<=2;dx++ { d:=max(abs(dx),abs(dy)); setFunc(cx+dx,cy+dy,d!=1) } } }
	drawFinder(3,3); drawFinder(size-4,3); drawFinder(3,size-4)
	for i:=8;i<size-8;i++ { setFunc(i,6,i%2==0); setFunc(6,i,i%2==0) }
	drawAlignment(26,26)
	formatData:=1<<3; rem:=formatData; for i:=0;i<10;i++ { rem=(rem<<1)^((rem>>9)*0x537) }; formatBits:=((formatData<<10)|rem)^0x5412
	getFormatBit:=func(i int)bool{return((formatBits>>i)&1)!=0}
	for i:=0;i<=5;i++{setFunc(8,i,getFormatBit(i))}; setFunc(8,7,getFormatBit(6)); setFunc(8,8,getFormatBit(7)); setFunc(7,8,getFormatBit(8)); for i:=9;i<15;i++{setFunc(14-i,8,getFormatBit(i))}; for i:=0;i<8;i++{setFunc(size-1-i,8,getFormatBit(i))}; for i:=8;i<15;i++{setFunc(8,size-15+i,getFormatBit(i))}; setFunc(8,size-8,true)
	dataBits:=make([]bool,0,len(code)*8); for _,b:=range code { for i:=7;i>=0;i-- { dataBits=append(dataBits,((b>>i)&1)!=0) } }
	bitIndex:=0
	for right:=size-1;right>=1;right-=2 { if right==6{right--}; for vert:=0;vert<size;vert++ { upward:=((right+1)&2)==0; y:=vert; if upward{y=size-1-vert}; for j:=0;j<2;j++ { x:=right-j; if function[y][x]{continue}; black:=false; if bitIndex<len(dataBits){black=dataBits[bitIndex];bitIndex++}; if(x+y)%2==0{black=!black}; if black{modules[y][x]=1}else{modules[y][x]=0} } } }
	out:=make([][]bool,size); for y:=range out { out[y]=make([]bool,size); for x:=range out[y] { out[y][x]=modules[y][x]==1 } }; return out,nil
}

func rsRemainder(data []byte,degree int)[]byte{gen:=[]byte{1};root:=byte(1);for i:=0;i<degree;i++{next:=make([]byte,len(gen)+1);for j,c:=range gen{next[j]^=c;next[j+1]^=gfMul(c,root)};gen=next;root=gfMul(root,2)};rem:=make([]byte,degree);for _,b:=range data{factor:=b^rem[0];copy(rem,rem[1:]);rem[degree-1]=0;for j:=0;j<degree;j++{rem[j]^=gfMul(gen[j+1],factor)}};return rem}
func gfMul(x,y byte)byte{var z byte;for i:=7;i>=0;i--{z=(z<<1)^((z>>7)*0x1D);if((y>>i)&1)!=0{z^=x}};return z}
func qrSVG(text string,scale int)(string,error){matrix,err:=qrV4L([]byte(text));if err!=nil{return"",err};if scale<2{scale=2};const border=4;n:=len(matrix);dim:=(n+border*2)*scale;var b strings.Builder;fmt.Fprintf(&b,`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" shape-rendering="crispEdges">`,dim,dim,dim,dim);b.WriteString(`<rect width="100%" height="100%" fill="white"/>`);b.WriteString(`<path d="`);for y:=0;y<n;y++{for x:=0;x<n;x++{if matrix[y][x]{px:=(x+border)*scale;py:=(y+border)*scale;fmt.Fprintf(&b,"M%d %dh%dv%dh-%dz",px,py,scale,scale,scale)}}};b.WriteString(`" fill="black"/></svg>`);return b.String(),nil}
func qrSVGError(message string)string{var b bytes.Buffer;b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 320 80"><rect width="100%" height="100%" fill="white"/><text x="10" y="32" font-family="sans-serif" font-size="14">`);b.WriteString(html.EscapeString(message));b.WriteString(`</text></svg>`);return b.String()}
func validateQRPayload(s string)error{if len([]byte(s))>78{return errors.New("QRコード化できる文字列は78バイトまでです")};return nil}
func abs(x int)int{if x<0{return-x};return x};func max(a,b int)int{if a>b{return a};return b}
